package client

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// EventListener defines the type for a callback function that processes incoming events.
type EventListener func(event *events.Event)

// EventSubscription represents a subscription to events in a specific instance.
type EventSubscription struct {
	subscriptionID      int32
	eventMapping        map[int]string
	activeSubscriptions types.Set[string]
	eventListener       EventListener
	Instance            string
	client              *YamcsClient
}

// CreateEventSubscription creates a new event subscription for a given instance.
func (client *YamcsClient) CreateEventSubscription(ctx context.Context, instance string) (*EventSubscription, error) {
	subscription, err := client.newEventSubscription(ctx, instance)
	if err != nil {
		return nil, err
	}

	client.subsMu.Lock()
	client.EventSubscriptions[subscription.subscriptionID] = subscription
	client.subsMu.Unlock()
	return subscription, nil
}

// FindEventSubscription returns the existing event subscription for the
// given instance, if one has already been created.
func (client *YamcsClient) FindEventSubscription(instance string) (*EventSubscription, bool) {
	client.subsMu.RLock()
	defer client.subsMu.RUnlock()
	for _, subscription := range client.EventSubscriptions {
		if subscription.Instance == instance {
			return subscription, true
		}
	}
	return nil, false
}

// HaltEventSubscriptionsForInstance halts and removes every event
// subscription registered for the given instance.
func (client *YamcsClient) HaltEventSubscriptionsForInstance(instance string) {
	client.subsMu.RLock()
	matches := make([]*EventSubscription, 0, 1)
	for _, subscription := range client.EventSubscriptions {
		if subscription.Instance == instance {
			matches = append(matches, subscription)
		}
	}
	client.subsMu.RUnlock()
	for _, subscription := range matches {
		subscription.Halt()
	}
}

// NewEventSubscription initializes a new EventSubscription and subscribes to events.
func (client *YamcsClient) newEventSubscription(ctx context.Context, instance string) (*EventSubscription, error) {
	cooldownKey := subscribeCooldownKey("events", instance, "")
	if err := client.checkSubscribeCooldown(cooldownKey); err != nil {
		return nil, err
	}

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

	_, callID, _, err := client.WebSocket.SendSync(ctx, message)
	client.recordSubscribeOutcome(ctx, cooldownKey, err)
	if err != nil {
		return nil, err
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// HandleEventMessage processes incoming server messages related to events.
func (client *YamcsClient) HandleEventMessage(message *api.ServerMessage) {
	event := &events.Event{}
	// Attempt to unmarshal the event data
	err := message.Data.UnmarshalTo(event)
	if err != nil {
		backend.Logger.Debug("Error unmarshalling event data", "error", err)
		return
	}

	// Retrieve the subscription using the call ID from the message
	callID := message.GetCall()
	client.subsMu.RLock()
	subscription, found := client.EventSubscriptions[callID]
	client.subsMu.RUnlock()
	if found && subscription.eventListener != nil {
		// Invoke the listener with the unmarshalled event data
		subscription.eventListener(event)
	}
}

// SetListener assigns an event listener to the subscription.
func (subscription *EventSubscription) SetListener(listener EventListener) {
	subscription.eventListener = listener
}

// Cancel subscription
func (subscription *EventSubscription) Halt() {

	subscription.client.subsMu.Lock()
	delete(subscription.client.EventSubscriptions, subscription.subscriptionID)
	subscription.client.subsMu.Unlock()

	// Prepare subscription request
	subscribeRequest := &api.CancelOptions{
		Call: subscription.subscriptionID,
	}

	anyMessage, _ := anypb.New(subscribeRequest)

	// Send the cancel request via WebSocket
	message := &api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	}

	subscription.client.WebSocket.Send(message)

}
