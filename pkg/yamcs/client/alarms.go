package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// ListAlarms retrieves alarms for a given instance and name, returning a paginated iterator.
func (c *YamcsClient) ListAlarms(ctx context.Context, instance, name string) *types.PaginatedRequestIterator[[]*alarms.AlarmData] {
	return types.NewPaginatedRequestIterator(c.HTTP, c.fetchAlarms(ctx, instance, name))
}

// fetchAlarms fetches a list of alarms from the Yamcs API.
func (c *YamcsClient) fetchAlarms(ctx context.Context, instance, name string) types.FetchFunction[[]*alarms.AlarmData] {
	return func() ([]*alarms.AlarmData, string, error) {
		response := &alarms.ListAlarmsResponse{}
		if err := c.HTTP.GetProto(ctx, fmt.Sprintf("/archive/%s/alarms/%s", instance, name), response); err != nil {
			return nil, "", err
		}
		return response.Alarms, response.GetContinuationToken(), nil
	}
}

// ListProcessorAlarms retrieves currently active alarms for a processor.
func (c *YamcsClient) ListProcessorAlarms(ctx context.Context, instance string, processor string) ([]*alarms.AlarmData, error) {
	response := &alarms.ListProcessorAlarmsResponse{}
	if err := c.HTTP.GetProto(ctx, fmt.Sprintf("/processors/%s/%s/alarms", instance, processor), response); err != nil {
		return nil, err
	}
	return response.Alarms, nil
}

// AcknowledgeAlarm acknowledges an alarm.
func (c *YamcsClient) AcknowledgeAlarm(ctx context.Context, instance string, processor string, alarmName string, seqNum uint32, comment string) error {
	request := &alarms.EditAlarmRequest{
		Instance:  stringPtr(instance),
		Processor: stringPtr(processor),
		Name:      &alarmName,
		Seqnum:    &seqNum,
		State:     stringPtr("acknowledged"),
		Comment:   &comment,
	}
	return c.HTTP.PatchProto(ctx, fmt.Sprintf("/processors/%s/%s/alarms/%s/%d", instance, processor, url.PathEscape(alarmName), seqNum), request, nil)
}

// ClearAlarm clears an alarm.
func (c *YamcsClient) ClearAlarm(ctx context.Context, instance string, processor string, alarmName string, seqNum uint32, comment string) error {
	request := &alarms.EditAlarmRequest{
		Instance:  stringPtr(instance),
		Processor: stringPtr(processor),
		Name:      &alarmName,
		Seqnum:    &seqNum,
		State:     stringPtr("cleared"),
		Comment:   &comment,
	}
	return c.HTTP.PatchProto(ctx, fmt.Sprintf("/processors/%s/%s/alarms/%s/%d", instance, processor, url.PathEscape(alarmName), seqNum), request, nil)
}

// ShelveAlarm shelves an alarm.
func (c *YamcsClient) ShelveAlarm(ctx context.Context, instance string, processor string, alarmName string, seqNum uint32, comment string, durationMs uint64) error {
	request := &alarms.EditAlarmRequest{
		Instance:       stringPtr(instance),
		Processor:      stringPtr(processor),
		Name:           &alarmName,
		Seqnum:         &seqNum,
		State:          stringPtr("shelved"),
		Comment:        &comment,
		ShelveDuration: &durationMs,
	}
	return c.HTTP.PatchProto(ctx, fmt.Sprintf("/processors/%s/%s/alarms/%s/%d", instance, processor, url.PathEscape(alarmName), seqNum), request, nil)
}

// UnshelveAlarm unshelves an alarm.
func (c *YamcsClient) UnshelveAlarm(ctx context.Context, instance string, processor string, alarmName string, seqNum uint32) error {
	request := &alarms.EditAlarmRequest{
		Instance:  stringPtr(instance),
		Processor: stringPtr(processor),
		Name:      &alarmName,
		Seqnum:    &seqNum,
	}
	// Yamcs uses the :unshelve action endpoint (similar to :acknowledge, :shelve, :clear)
	return c.HTTP.PostProto(ctx, fmt.Sprintf("/processors/%s/%s/alarms/%s/%d:unshelve", instance, processor, url.PathEscape(alarmName), seqNum), request, nil)
}

func stringPtr(s string) *string {
	return &s
}

// AlarmListener is a function that handles incoming alarm events.
type AlarmListener func(event *alarms.AlarmData) error

// AlarmSubscription represents a subscription to Yamcs alarm events.
type AlarmSubscription struct {
	callID   int32
	listener AlarmListener
	instance string
	client   *YamcsClient
}

// CreateAlarmSubscription initializes a new alarm subscription.
func (c *YamcsClient) CreateAlarmSubscription(ctx context.Context, instance string, processor string) (*AlarmSubscription, error) {
	return c.newAlarmSubscription(ctx, instance, processor)
}

// newAlarmSubscription handles the subscription logic for alarms.
func (c *YamcsClient) newAlarmSubscription(ctx context.Context, instance string, processor string) (*AlarmSubscription, error) {
	subscription := &AlarmSubscription{
		client:   c,
		instance: instance,
	}

	subscribeRequest := &alarms.SubscribeAlarmsRequest{
		Instance:  new(instance),
		Processor: new(processor),
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "alarms",
		Options: anyMessage,
	}

	_, callID, _, err := c.WebSocket.SendSync(ctx, message)
	if err != nil {
		return nil, err
	}

	subscription.callID = callID
	c.AlarmSubscriptions[callID] = subscription
	return subscription, nil
}

// HandleAlarmMessage listens for incoming alarm events.
func (c *YamcsClient) HandleAlarmMessage(msg *api.ServerMessage) {
	if msg.GetType() != "alarms" {
		return
	}

	alarmData := &alarms.AlarmData{}
	if err := msg.Data.UnmarshalTo(alarmData); err != nil {
		backend.Logger.Debug("Error unmarshaling Yamcs alarm message", "error", err)
		return
	}

	if subscription, exists := c.AlarmSubscriptions[msg.GetCall()]; exists && subscription.listener != nil {
		subscription.listener(alarmData)
	}
}

// SetListener assigns a callback function to an AlarmSubscription.
func (sub *AlarmSubscription) SetListener(listener AlarmListener) {
	sub.listener = listener
}

// GetInstance returns the instance name for this alarm subscription.
func (sub *AlarmSubscription) GetInstance() string {
	return sub.instance
}

// Halt cancels the alarm subscription.
func (sub *AlarmSubscription) Halt() {
	delete(sub.client.AlarmSubscriptions, sub.callID)

	cancelRequest := &api.CancelOptions{
		Call: sub.callID,
	}

	anyMessage, _ := anypb.New(cancelRequest)
	sub.client.WebSocket.Send(&api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	})
}

// GlobalStatusListener is a function that handles global alarm status events.
type GlobalStatusListener func(event *alarms.GlobalAlarmStatus)

// GlobalStatusSubscription represents a subscription to global alarm status events.
type GlobalStatusSubscription struct {
	callID              int32
	eventMapping        map[int]string
	subscribedInstances types.Set[string]
	listener            GlobalStatusListener
	instance            string
	client              *YamcsClient
}

// CreateGlobalAlarmStatusSubscription initializes a global alarm status subscription.
func (c *YamcsClient) CreateGlobalAlarmStatusSubscription(ctx context.Context, instance string, processor string) (*GlobalStatusSubscription, error) {
	return c.newGlobalAlarmStatusSubscription(ctx, instance, processor)
}

// newGlobalAlarmStatusSubscription handles the subscription logic for global alarm status updates.
func (c *YamcsClient) newGlobalAlarmStatusSubscription(ctx context.Context, instance string, processor string) (*GlobalStatusSubscription, error) {
	subscription := &GlobalStatusSubscription{
		client:              c,
		instance:            instance,
		eventMapping:        make(map[int]string),
		subscribedInstances: types.Set[string]{},
	}

	subscribeRequest := &alarms.SubscribeGlobalStatusRequest{
		Instance:  new(instance),
		Processor: new(processor),
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "global-alarm-status",
		Options: anyMessage,
	}

	_, callID, _, err := c.WebSocket.SendSync(context.Background(), message)
	if err != nil {
		return nil, err
	}

	subscription.callID = callID
	c.GlobalAlarmStatusSubscriptions[callID] = subscription
	return subscription, nil
}

// HandleGlobalStatusMessage listens for global alarm status events.
func (c *YamcsClient) HandleGlobalStatusMessage(msg *api.ServerMessage) {
	if msg.GetType() != "global-alarm-status" {
		return
	}

	statusData := &alarms.GlobalAlarmStatus{}
	if err := msg.Data.UnmarshalTo(statusData); err != nil {
		backend.Logger.Debug("Error unmarshaling Yamcs global alarm status message", "error", err)
		return
	}

	if subscription, exists := c.GlobalAlarmStatusSubscriptions[msg.GetCall()]; exists && subscription.listener != nil {
		subscription.listener(statusData)
	}
}

// SetListener assigns a callback function to a GlobalStatusSubscription.
func (sub *GlobalStatusSubscription) SetListener(listener GlobalStatusListener) {
	sub.listener = listener
}

// GetInstance returns the instance name for this global alarm status subscription.
func (sub *GlobalStatusSubscription) GetInstance() string {
	return sub.instance
}

// Halt stops the global alarm status subscription and removes it from the client.
func (sub *GlobalStatusSubscription) Halt() {
	delete(sub.client.GlobalAlarmStatusSubscriptions, sub.callID)

	cancelRequest := &api.CancelOptions{
		Call: sub.callID,
	}

	anyMessage, _ := anypb.New(cancelRequest)
	sub.client.WebSocket.Send(&api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	})
}
