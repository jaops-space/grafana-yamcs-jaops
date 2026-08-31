package source

import (
	"context"
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

func (ep *YamcsEndpoint) RequestAlarmsStream(ctx context.Context, path string) error {

	if _, err := ep.getOrCreateAlarmsSubscription(ctx); err != nil {
		return err
	}
	if _, err := ep.getOrCreateGlobalAlarmStatusSubscription(ctx); err != nil {
		return err
	}

	ep.mu.Lock()
	ep.Alarms[path] = make([]*alarms.AlarmData, 0)
	ep.AlarmSignals[path] = make(chan struct{}, StreamSignalBufferSize)
	cacheEmpty := len(ep.AlarmCache) == 0
	ep.mu.Unlock()

	if cacheEmpty {
		cli, err := ep.GetClient()
		if err != nil {
			return err
		}
		alarmList, err := cli.ListProcessorAlarms(ctx, ep.GetInstanceName(), ep.GetProcessorName())
		if err != nil {
			return err
		}
		ep.mu.Lock()
		for _, alarm := range alarmList {
			// Skip cleared alarms when loading initial cache
			if alarm.GetClearInfo() != nil {
				continue
			}
			if alarm.GetId() == nil {
				continue
			}
			qualifiedName := alarm.GetId().GetNamespace() + "/" + alarm.GetId().GetName()
			alarmID := fmt.Sprintf("%s/%d", qualifiedName, alarm.GetSeqNum())
			ep.AlarmCache[alarmID] = alarm
		}
		ep.mu.Unlock()
	}
	return nil
}

// getOrCreateAlarmsSubscription does not touch mu - subscription
// lookup/creation is guarded by the client's own subsMu (for the map) and
// alarmSubscribeMu (for the create race), so a slow/stuck subscribe attempt
// never blocks unrelated endpoint state guarded by mu.
func (ep *YamcsEndpoint) getOrCreateAlarmsSubscription(ctx context.Context) (*client.AlarmSubscription, error) {

	cli, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := cli.FindAlarmSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}

	ep.alarmSubscribeMu.Lock()
	defer ep.alarmSubscribeMu.Unlock()

	if subscription, found := cli.FindAlarmSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}
	subscription, err := cli.CreateAlarmSubscription(ctx, ep.GetInstanceName(), ep.GetProcessorName())
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.getAlarmsListener())
	return subscription, nil
}

// getOrCreateGlobalAlarmStatusSubscription does not touch mu for the same
// reason as getOrCreateAlarmsSubscription above.
func (ep *YamcsEndpoint) getOrCreateGlobalAlarmStatusSubscription(ctx context.Context) (*client.GlobalStatusSubscription, error) {

	cli, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := cli.FindGlobalAlarmStatusSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}

	ep.alarmSubscribeMu.Lock()
	defer ep.alarmSubscribeMu.Unlock()

	if subscription, found := cli.FindGlobalAlarmStatusSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}
	subscription, err := cli.CreateGlobalAlarmStatusSubscription(ctx, ep.GetInstanceName(), ep.GetProcessorName())
	if err != nil {
		return nil, err
	}
	subscription.SetListener(func(status *alarms.GlobalAlarmStatus) {
		ep.mu.Lock()
		defer ep.mu.Unlock()
		ep.GlobalAlarmStatus = status
		for path := range ep.Alarms {
			ep.NotifyAlarmsStream(path)
		}
	})
	return subscription, nil
}

// GetGlobalAlarmStatus returns a consistent snapshot of GlobalAlarmStatus under the read lock.
func (ep *YamcsEndpoint) GetGlobalAlarmStatus() *alarms.GlobalAlarmStatus {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.GlobalAlarmStatus
}

// TODO: make alarm ordering and concatenation client-side
func (ep *YamcsEndpoint) GetAlarmsStream(path string) []*alarms.AlarmData {
	// Return all cached alarms (complete list of active alarms)
	ep.mu.RLock()
	result := make([]*alarms.AlarmData, 0, len(ep.AlarmCache))
	for _, alarm := range ep.AlarmCache {
		result = append(result, alarm)
	}
	ep.mu.RUnlock()

	// Sort alarms consistently to prevent UI reordering
	// Sort by: 1) Trigger time (newest first), 2) Qualified name, 3) SeqNum
	sort.Slice(result, func(i, j int) bool {
		timeI := result[i].GetTriggerTime().AsTime()
		timeJ := result[j].GetTriggerTime().AsTime()

		// Sort by trigger time (newest first)
		if !timeI.Equal(timeJ) {
			return timeI.After(timeJ)
		}

		// If same time, sort by qualified name
		nameI := result[i].GetId().GetNamespace() + "/" + result[i].GetId().GetName()
		nameJ := result[j].GetId().GetNamespace() + "/" + result[j].GetId().GetName()
		if nameI != nameJ {
			return nameI < nameJ
		}

		// If same name, sort by sequence number
		return result[i].GetSeqNum() < result[j].GetSeqNum()
	})

	return result
}

func (ep *YamcsEndpoint) ClearAlarmsStream(path string) {
	// Clear only the update buffer, not the cache
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.Alarms[path] = make([]*alarms.AlarmData, 0)
}

func (ep *YamcsEndpoint) NotifyAlarmsStream(path string) {
	if signal, ok := ep.AlarmSignals[path]; ok {
		select {
		case signal <- struct{}{}:
		default:
			backend.Logger.Warn("dropping alarm notification because stream buffer is full", "path", path)
		}
	}
}

func (ep *YamcsEndpoint) GetAlarmsSignal(path string) <-chan struct{} {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.AlarmSignals[path]
}

func (ep *YamcsEndpoint) DrainAlarmsSignal(signal <-chan struct{}) {
	for {
		select {
		case _, ok := <-signal:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func (ep *YamcsEndpoint) WithdrawAlarmsStreamRequest(path string) error {

	ep.mu.Lock()
	defer ep.mu.Unlock()
	delete(ep.Alarms, path)
	if signal, ok := ep.AlarmSignals[path]; ok {
		close(signal)
		delete(ep.AlarmSignals, path)
	}
	if len(ep.Alarms) == 0 {
		c, err := ep.GetClient()
		if err != nil {
			return err
		}
		c.HaltAlarmSubscriptionsForInstance(ep.GetInstanceName())
		c.HaltGlobalAlarmStatusSubscriptionsForInstance(ep.GetInstanceName())
	}
	return nil
}

// GetAlarmsListener returns a function that listens for alarm events from a specific Yamcs instance.
func (ep *YamcsEndpoint) getAlarmsListener() client.AlarmListener {
	return func(alarm *alarms.AlarmData) error {
		hasUpdate := false

		// Generate unique alarm ID (namespace/name/seqNum)
		if alarm.GetId() == nil {
			return nil
		}
		qualifiedName := alarm.GetId().GetNamespace() + "/" + alarm.GetId().GetName()
		alarmID := fmt.Sprintf("%s/%d", qualifiedName, alarm.GetSeqNum())

		ep.mu.Lock()
		defer ep.mu.Unlock()
		if shouldRemoveAlarmFromCache(alarm) {
			delete(ep.AlarmCache, alarmID)
			hasUpdate = true
		} else {
			ep.AlarmCache[alarmID] = alarm
			hasUpdate = true
		}

		if hasUpdate {
			for path := range ep.Alarms {
				ep.NotifyAlarmsStream(path)
			}
		}
		return nil
	}
}

func shouldRemoveAlarmFromCache(alarm *alarms.AlarmData) bool {
	if alarm.GetClearInfo() != nil {
		return true
	}

	return alarm.GetProcessOK() && !alarm.GetTriggered() && alarm.GetAcknowledged()
}
