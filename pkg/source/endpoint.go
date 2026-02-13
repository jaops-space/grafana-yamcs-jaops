package source

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// YamcsEndpoint represents an endpoint for Yamcs communication.
type YamcsEndpoint struct {
	Multiplexer *Multiplexer

	ID             string
	Instance       client.Instance
	Processor      client.Processor
	Parameters     map[string]*ParameterDemand
	Events         map[string][]*events.Event
	CommandHistory map[string][]*commanding.CommandHistoryEntry
	Alarms         map[string][]*alarms.AlarmData
	AlarmCache     map[string]*alarms.AlarmData // Cache of all active alarms by ID
	GlobalAlarmStatus *alarms.GlobalAlarmStatus

	CurrentTime time.Time
}

// ParameterDemand represents a demand for a specific parameter.
type ParameterDemand struct {
	endpoint *YamcsEndpoint

	LastReceived time.Time
	Name         string
	Unit         string
	Thresholds   []*data.Threshold
	Streams      map[string]*ParameterStreamDemand
}

// ParameterStreamDemand represents a demand for a specific parameter stream.
type ParameterStreamDemand struct {
	parameter *ParameterDemand

	Path   string
	Buffer []client.ParameterValue
}

func (ep *YamcsEndpoint) RequestTime() {
	client := ep.GetClient()
	if client.HasTimeSubscriptionFor(ep.Instance, ep.Processor) {
		return
	}
	subscription, err := client.CreateTimeSubscription(ep.Instance, ep.Processor)
	if err != nil {
		backend.Logger.Error(err.Error())
		return
	}
	subscription.SetTimeListener(ep.GetTimeHandler())
}

func (ep *YamcsEndpoint) GetTimeHandler() func(t time.Time) {
	return func(currentTime time.Time) {
		ep.CurrentTime = currentTime
		backend.Logger.Debug("Updating time", "time", currentTime)
	}
}

// GetHostConfiguration retrieves the host configuration for the endpoint.
func (ep *YamcsEndpoint) GetHostConfiguration(name string) *config.YamcsHostConfiguration {
	return ep.Multiplexer.Configuration.Hosts[ep.Multiplexer.Configuration.Endpoints[ep.ID].Host]
}

// GetConfiguration retrieves the source configuration for the endpoint.
func (ep *YamcsEndpoint) GetConfiguration() *config.YamcsEndpointConfiguration {
	return ep.Multiplexer.Configuration.Endpoints[ep.ID]
}

// GetParameterDemand retrieves or initializes a ParameterDemand.
func (ep *YamcsEndpoint) GetParameterDemand(parameter string) *ParameterDemand {

	if ep.Parameters[parameter] == nil {
		client := ep.GetClient()
		unit := ""
		thresholds := make([]*data.Threshold, 0)

		paramInfo, err := client.GetParameter(ep.Instance, parameter)
		if err == nil {
			paramType := paramInfo.GetType()
			unitSet := paramType.GetUnitSet()
			thresholds = tools.ConvertAlarmInfoToThresholds(paramType.GetDefaultAlarm())
			if len(unitSet) > 0 {
				unit = unitSet[0].GetUnit()
			}
		}

		ep.Parameters[parameter] = &ParameterDemand{
			endpoint:   ep,
			Name:       parameter,
			Unit:       unit,
			Thresholds: thresholds,
			Streams:    make(map[string]*ParameterStreamDemand),
		}
	}
	return ep.Parameters[parameter]
}

// GetChannelParameterListener returns a function to listen for parameter updates.
func (ep *YamcsEndpoint) GetChannelParameterListener() func(parameter string, value *pvalue.ParameterValue) {
	return func(parameter string, value *pvalue.ParameterValue) {

		paramDemand := ep.GetParameterDemand(parameter)
		streamDemands := paramDemand.Streams
		paramDemand.LastReceived = time.Now()

		if value.GetAcquisitionStatus() != pvalue.AcquisitionStatus_ACQUIRED {
			backend.Logger.Debug("Ignoring parameter value", "parameter", parameter, "status", value.GetAcquisitionStatus())
			return
		}

		for _, streamDemand := range streamDemands {
			streamDemand.Buffer = append(streamDemand.Buffer, value)
		}

	}
}

// RequestNewParameterStream adds a new parameter stream to the endpoint.
func (ep *YamcsEndpoint) RequestNewParameterStream(name string, path string) error {

	ep.GetParameterDemand(name)

	ep.Parameters[name].Streams[path] = &ParameterStreamDemand{
		parameter: ep.Parameters[name],
		Path:      path,
		Buffer:    make([]*pvalue.ParameterValue, 0),
	}

	subscription, err := ep.GetParameterSubscription()
	if err != nil {
		backend.Logger.Error("Error getting parameter subscription", "error", err)
		return err
	}

	if !subscription.Has(name) {
		backend.Logger.Debug("Adding parameter to subscription", "parameter", name)
		subscription.Add(name)
	}
	backend.Logger.Debug("Current subscriptions", "subscriptions")
	for name := range subscription.ActiveSubscriptions {
		backend.Logger.Debug(name)
	}

	return nil
}

// GetParameterStreamBuffer retrieves the buffer for a specific parameter stream.
func (ep *YamcsEndpoint) GetParameterStreamBuffer(parameter string, path string) []client.ParameterValue {
	if ep.Parameters[parameter].Streams[path] == nil {
		ep.RequestNewParameterStream(parameter, path)
		return ep.Parameters[parameter].Streams[path].Buffer
	}
	return ep.Parameters[parameter].Streams[path].Buffer
}

// ClearParameterStream clears the buffer for a specific parameter stream.
func (ep *YamcsEndpoint) ClearParameterStream(parameter string, path string) {
	if ep.Parameters[parameter].Streams[path] == nil {
		ep.RequestNewParameterStream(parameter, path)
	}
	ep.Parameters[parameter].Streams[path].Buffer = make([]client.ParameterValue, 0)
}

// WithdrawParameterStreamRequest removes a parameter stream request.
func (ep *YamcsEndpoint) WithdrawParameterStreamRequest(name string, path string) error {

	ep.GetParameterDemand(name)
	client := ep.GetClient()

	delete(ep.Parameters[name].Streams, path)
	if len(ep.Parameters[name].Streams) == 0 && client != nil && client.IsWebSocketConnected() {
		subscription, err := ep.GetParameterSubscription()
		if err != nil {
			return err
		}
		subscription.Remove(name)
	}
	return nil
}

// GetClient retrieves the Yamcs client for this endpoint.
func (ep *YamcsEndpoint) GetClient() *client.YamcsClient {
	yamcsClient, err := ep.Multiplexer.ConnMgr.GetClient(ep.GetConfiguration().Host)
	if err != nil {
		return nil
	}
	return yamcsClient
}

// GetParameterSubscription retrieves or creates a parameter subscription.
func (ep *YamcsEndpoint) GetParameterSubscription() (*client.ParameterSubscription, error) {
	client := ep.GetClient()
	for _, subscription := range client.ParameterSubscriptions {
		if subscription.Instance == ep.Instance.GetName() && subscription.Processor == ep.Processor.GetName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateParameterSubscription(ep.Instance, ep.Processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.GetChannelParameterListener())
	return subscription, nil
}

// RequestEventsStream initiates an event stream subscription.
func (ep *YamcsEndpoint) RequestEventsStream(path string) {
	ep.GetEventsSubscription()
	ep.Events[path] = make([]*events.Event, 0)
}

func (source *YamcsEndpoint) GetEventsSubscription() (*client.EventSubscription, error) {

	client := source.GetClient()
	for _, subscription := range client.EventSubscriptions {
		if subscription.Instance == source.Instance.GetName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateEventSubscription(source.Instance)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(source.Multiplexer.GetEventListener(source.Instance))
	return subscription, nil

}

func (ep *YamcsEndpoint) GetEventsStream(path string) []*events.Event {

	return ep.Events[path]

}

func (ep *YamcsEndpoint) ClearEventsStream(path string) {

	ep.Events[path] = make([]*events.Event, 0)

}

// WithdrawEventsStreamRequest stops an event stream subscription.
func (ep *YamcsEndpoint) WithdrawEventsStreamRequest(path string) {

	delete(ep.Events, path)
	if len(ep.Events) == 0 {
		client := ep.GetClient()
		for _, subscription := range client.EventSubscriptions {
			if subscription.Instance == ep.Instance.GetName() {
				subscription.Halt()
			}
		}
	}
}

/**

COMMAND HISTORY

**/

func (ep *YamcsEndpoint) RequestCommandHistoryStream(path string) {
	ep.GetCommandHistorySubscription()
	ep.CommandHistory[path] = make([]*commanding.CommandHistoryEntry, 0)
}

func (ep *YamcsEndpoint) GetCommandHistorySubscription() (*client.CommandHistorySubscription, error) {
	client := ep.GetClient()
	for _, subscription := range client.CommandHistorySubscriptions {
		if subscription.Instance == ep.Instance.GetName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateCommandHistorySubscription(ep.Instance, ep.Processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.Multiplexer.GetCommandHistoryListener(ep.Instance))
	return subscription, nil
}

func (ep *YamcsEndpoint) GetCommandHistoryStream(path string) []*commanding.CommandHistoryEntry {
	return ep.CommandHistory[path]
}

func (ep *YamcsEndpoint) ClearCommandHistoryStream(path string) {
	ep.CommandHistory[path] = make([]*commanding.CommandHistoryEntry, 0)
}

func (ep *YamcsEndpoint) WithdrawCommandHistoryStreamRequest(path string) {
	delete(ep.CommandHistory, path)
	if len(ep.CommandHistory) == 0 {
		client := ep.GetClient()
		for _, subscription := range client.CommandHistorySubscriptions {
			if subscription.Instance == ep.Instance.GetName() {
				subscription.Halt()
			}
		}
	}
}

/**

Alarms

**/

func (ep *YamcsEndpoint) RequestAlarmsStream(path string) {
	ep.GetAlarmsSubscription()
	ep.GetGlobalAlarmStatusSubscription()
	ep.Alarms[path] = make([]*alarms.AlarmData, 0)

	// Load initial alarms into cache if cache is empty
	if len(ep.AlarmCache) == 0 {
		yamcs := ep.GetClient()
		alarmList, err := yamcs.ListProcessorAlarms(ep.Instance, ep.Processor)
		if err == nil {
			for _, alarm := range alarmList {
				qualifiedName := alarm.GetId().GetNamespace() + "/" + alarm.GetId().GetName()
				alarmID := fmt.Sprintf("%s/%d", qualifiedName, alarm.GetSeqNum())
				ep.AlarmCache[alarmID] = alarm
			}
		}
	}
}

func (ep *YamcsEndpoint) GetAlarmsSubscription() (*client.AlarmSubscription, error) {
	c := ep.GetClient()
	for _, subscription := range c.AlarmSubscriptions {
		if subscription.GetInstance() == ep.Instance.GetName() {
			return subscription, nil
		}
	}
	subscription, err := c.CreateAlarmSubscription(ep.Instance, ep.Processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.Multiplexer.GetAlarmsListener(ep.Instance))
	return subscription, nil
}

func (ep *YamcsEndpoint) GetGlobalAlarmStatusSubscription() (*client.GlobalStatusSubscription, error) {
	c := ep.GetClient()
	for _, subscription := range c.GlobalAlarmStatusSubscriptions {
		if subscription.GetInstance() == ep.Instance.GetName() {
			return subscription, nil
		}
	}
	subscription, err := c.CreateGlobalAlarmStatusSubscription(ep.Instance, ep.Processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(func(status *alarms.GlobalAlarmStatus) {
		ep.GlobalAlarmStatus = status
	})
	return subscription, nil
}

func (ep *YamcsEndpoint) GetAlarmsStream(path string) []*alarms.AlarmData {
	// Return all cached alarms (complete list of active alarms)
	result := make([]*alarms.AlarmData, 0, len(ep.AlarmCache))
	for _, alarm := range ep.AlarmCache {
		result = append(result, alarm)
	}

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
	ep.Alarms[path] = make([]*alarms.AlarmData, 0)
}

func (ep *YamcsEndpoint) WithdrawAlarmsStreamRequest(path string) {
	delete(ep.Alarms, path)
	if len(ep.Alarms) == 0 {
		c := ep.GetClient()
		for _, subscription := range c.AlarmSubscriptions {
			if subscription.GetInstance() == ep.Instance.GetName() {
				subscription.Halt()
			}
		}
	}
}
