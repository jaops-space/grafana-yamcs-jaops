package client

import (
	"fmt"
	"log"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// ListAlarms retrieves alarms for a given instance and name, returning a paginated iterator.
func (c *YamcsClient) ListAlarms(instance, name string) *types.PaginatedRequestIterator[[]*alarms.AlarmData] {
	return types.NewPaginatedRequestIterator(c.HTTP, c.fetchAlarms(instance, name))
}

// fetchAlarms fetches a list of alarms from the Yamcs API.
func (c *YamcsClient) fetchAlarms(instance, name string) types.FetchFunction[[]*alarms.AlarmData] {
	return func() ([]*alarms.AlarmData, string, error) {
		response := &alarms.ListAlarmsResponse{}
		if err := c.HTTP.GetProto(fmt.Sprintf("/archive/%s/alarms/%s", instance, name), response); err != nil {
			return nil, "", err
		}
		return response.Alarms, response.GetContinuationToken(), nil
	}
}

// AlarmListener is a function that handles incoming alarm events.
type AlarmListener func(event *alarms.AlarmData)

// AlarmSubscription represents a subscription to Yamcs alarm events.
type AlarmSubscription struct {
	callID   int
	listener AlarmListener
	instance string
	client   *YamcsClient
}

// CreateAlarmSubscription initializes a new alarm subscription.
func (c *YamcsClient) CreateAlarmSubscription(instance Instance, processor Processor) (*AlarmSubscription, error) {
	return c.newAlarmSubscription(instance, processor)
}

// newAlarmSubscription handles the subscription logic for alarms.
func (c *YamcsClient) newAlarmSubscription(instance Instance, processor Processor) (*AlarmSubscription, error) {
	subscription := &AlarmSubscription{
		client:   c,
		instance: instance.GetName(),
	}

	subscribeRequest := &alarms.SubscribeAlarmsRequest{
		Instance:  instance.Name,
		Processor: processor.Name,
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "alarms",
		Options: anyMessage,
	}

	_, callID, _, err := c.WebSocket.SendSync(message)
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
		log.Println("Error unmarshaling Yamcs alarm message:", err)
		return
	}

	if subscription, exists := c.AlarmSubscriptions[int(msg.GetCall())]; exists && subscription.listener != nil {
		subscription.listener(alarmData)
	}
}

// SetListener assigns a callback function to an AlarmSubscription.
func (sub *AlarmSubscription) SetListener(listener AlarmListener) {
	sub.listener = listener
}

// GlobalStatusListener is a function that handles global alarm status events.
type GlobalStatusListener func(event *alarms.GlobalAlarmStatus)

// GlobalStatusSubscription represents a subscription to global alarm status events.
type GlobalStatusSubscription struct {
	callID              int
	eventMapping        map[int]string
	subscribedInstances types.Set[string]
	listener            GlobalStatusListener
	instance            string
	client              *YamcsClient
}

// CreateGlobalAlarmStatusSubscription initializes a global alarm status subscription.
func (c *YamcsClient) CreateGlobalAlarmStatusSubscription(instance Instance, processor Processor) (*GlobalStatusSubscription, error) {
	return c.newGlobalAlarmStatusSubscription(instance, processor)
}

// newGlobalAlarmStatusSubscription handles the subscription logic for global alarm status updates.
func (c *YamcsClient) newGlobalAlarmStatusSubscription(instance Instance, processor Processor) (*GlobalStatusSubscription, error) {
	subscription := &GlobalStatusSubscription{
		client:              c,
		instance:            instance.GetName(),
		eventMapping:        make(map[int]string),
		subscribedInstances: types.Set[string]{},
	}

	subscribeRequest := &alarms.SubscribeGlobalStatusRequest{
		Instance:  instance.Name,
		Processor: processor.Name,
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "global-alarm-status",
		Options: anyMessage,
	}

	_, callID, _, err := c.WebSocket.SendSync(message)
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
		log.Println("Error unmarshaling Yamcs global alarm status message:", err)
		return
	}

	if subscription, exists := c.GlobalAlarmStatusSubscriptions[int(msg.GetCall())]; exists && subscription.listener != nil {
		subscription.listener(statusData)
	}
}

// SetListener assigns a callback function to a GlobalStatusSubscription.
func (sub *GlobalStatusSubscription) SetListener(listener GlobalStatusListener) {
	sub.listener = listener
}
