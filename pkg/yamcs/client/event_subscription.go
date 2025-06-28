package client

import (
	"log"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// EventListener defines the type for a callback function that processes incoming events.
type EventListener func(event *events.Event)

// EventSubscription represents a subscription to events in a specific instance.
type EventSubscription struct {
	subscriptionID      int
	eventMapping        map[int]string
	activeSubscriptions types.Set[string]
	eventListener       EventListener
	Instance            string
	client              *YamcsClient
}

// CreateEventSubscription creates a new event subscription for a given instance.
func (client *YamcsClient) CreateEventSubscription(instance Instance) (*EventSubscription, error) {
	subscription, err := client.newEventSubscription(instance.GetName())
	if err != nil {
		return nil, err
	}

	client.EventSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// NewEventSubscription initializes a new EventSubscription and subscribes to events.
func (client *YamcsClient) newEventSubscription(instance string) (*EventSubscription, error) {
	subscription := &EventSubscription{
		client:              client,
		Instance:            instance,
		eventMapping:        make(map[int]string),
		activeSubscriptions: types.Set[string]{},
	}

	// Prepare subscription request
	subscribeRequest := &events.SubscribeEventsRequest{
		Instance: &instance,
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	// Send the subscription request via WebSocket
	message := &api.ClientMessage{
		Type:    "events",
		Options: anyMessage,
	}

	_, callID, _, err := client.WebSocket.SendSync(message)
	if err != nil {
		return nil, err
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// HandleEventMessage processes incoming server messages related to events.
func (client *YamcsClient) HandleEventMessage(message *api.ServerMessage) {
	// Check if the message type is "events"
	if message.GetType() == "events" {
		event := &events.Event{}
		// Attempt to unmarshal the event data
		err := message.Data.UnmarshalTo(event)
		if err != nil {
			log.Default().Printf("Error unmarshalling event data: %v\n", err)
			return
		}

		// Retrieve the subscription using the call ID from the message
		callID := message.GetCall()
		subscription, found := client.EventSubscriptions[int(callID)]
		if found && subscription.eventListener != nil {
			// Invoke the listener with the unmarshalled event data
			subscription.eventListener(event)
		}
	}
}

// SetListener assigns an event listener to the subscription.
func (subscription *EventSubscription) SetListener(listener EventListener) {
	subscription.eventListener = listener
}

// Cancel subscription
func (subscription *EventSubscription) Halt() {

	delete(subscription.client.EventSubscriptions, subscription.subscriptionID)

	// Prepare subscription request
	subscribeRequest := &api.CancelOptions{
		Call: int32(subscription.subscriptionID),
	}

	anyMessage, _ := anypb.New(subscribeRequest)

	// Send the cancel request via WebSocket
	message := &api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	}

	subscription.client.WebSocket.SendSync(message)

}
