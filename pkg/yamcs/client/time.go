package client

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	ptime "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/time"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TimeListener func(currentTime time.Time)

// TimeSubscription manages a subscription to a set of parameters from a Yamcs instance and processor.
type TimeSubscription struct {
	subscriptionID int32
	Instance       string
	Processor      string
	listenersMu    sync.Mutex // guards listeners: appended to by AddTimeListener/SetTimeListener (called from endpoint setup goroutines) and ranged over by notifyListeners (called from the WebSocket's read-loop goroutine)
	listeners      []TimeListener
	client         *YamcsClient
}

// CreateTimeSubscription creates a new time subscription with an initial
// listener already wired into subscription.listeners before the
// subscription is published to client.TimeSubscriptions, so
// HandleTimeMessage can never dispatch to a listener-less subscription.
// Additional listeners can be attached later via AddTimeListener.
func (client *YamcsClient) CreateTimeSubscription(ctx context.Context, instance string, processor string, listener TimeListener) (*TimeSubscription, error) {

	subscription, err := client.newTimeSubscription(ctx, instance, processor, listener)
	if err != nil {
		return nil, err
	}

	client.subsMu.Lock()
	client.TimeSubscriptions[subscription.subscriptionID] = subscription
	client.subsMu.Unlock()
	return subscription, nil

}

// SubscribeTime subscribes to time updates from a specific instance and processor.
func (client *YamcsClient) newTimeSubscription(ctx context.Context, instance string, processor string, listener TimeListener) (*TimeSubscription, error) {
	cooldownKey := subscribeCooldownKey("time", instance, processor)
	if err := client.checkSubscribeCooldown(cooldownKey); err != nil {
		return nil, err
	}

	// Create the subscription request for time updates
	subscribeTimeRequest := &ptime.SubscribeTimeRequest{
		Instance:  &instance,
		Processor: &processor,
	}

	// Convert the subscription request into an Any message
	anyMessage, err := anypb.New(subscribeTimeRequest)
	if err != nil {
		return nil, err
	}

	// Prepare the message to send via WebSocket
	message := &api.ClientMessage{
		Type:    "time",     // Message type indicating it's a time subscription
		Id:      32,         // Unique message identifier
		Options: anyMessage, // Attach the Any message containing the subscription request
	}

	_, callID, _, err := client.WebSocket.SendSync(ctx, message)
	client.recordSubscribeOutcome(ctx, cooldownKey, err)
	if err != nil {
		return nil, err
	}

	subscription := &TimeSubscription{
		subscriptionID: callID,
		Instance:       instance,
		Processor:      processor,
		listeners:      make([]TimeListener, 0, 1),
		client:         client,
	}
	if listener != nil {
		subscription.listeners = append(subscription.listeners, listener)
	}

	backend.Logger.Debug("subscribing to processor time", "proc", processor)

	return subscription, nil
}

func (subscription *TimeSubscription) Halt() {

	subscription.client.subsMu.Lock()
	delete(subscription.client.TimeSubscriptions, subscription.subscriptionID)
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

func (client *YamcsClient) HandleTimeMessage(message *api.ServerMessage) {

	timestamp := &timestamppb.Timestamp{}
	if err := message.Data.UnmarshalTo(timestamp); err != nil {
		backend.Logger.Error("Error unmarshalling time subscription message", "error", exception.Wrap("Unmarshal error", "SUBSCRIPTION_UNMARSHALL_ERROR", err))
		return
	}

	// Retrieve the subscription by call ID
	callID := message.GetCall()
	client.subsMu.RLock()
	subscription, found := client.TimeSubscriptions[callID]
	client.subsMu.RUnlock()
	if !found {
		return
	}

	subscription.notifyListeners(timestamp.AsTime())
}

func (subscription *TimeSubscription) SetTimeListener(listener TimeListener) {
	subscription.listenersMu.Lock()
	defer subscription.listenersMu.Unlock()
	subscription.listeners = []TimeListener{listener}
}

func (subscription *TimeSubscription) AddTimeListener(listener TimeListener) {
	subscription.listenersMu.Lock()
	defer subscription.listenersMu.Unlock()
	subscription.listeners = append(subscription.listeners, listener)
}

func (subscription *TimeSubscription) notifyListeners(currentTime time.Time) {
	subscription.listenersMu.Lock()
	listeners := make([]TimeListener, len(subscription.listeners))
	copy(listeners, subscription.listeners)
	subscription.listenersMu.Unlock()

	for _, listener := range listeners {
		if listener != nil {
			listener(currentTime)
		}
	}
}

func (client *YamcsClient) GetTimeSubscription(instance string, processor string) (*TimeSubscription, bool) {
	client.subsMu.RLock()
	defer client.subsMu.RUnlock()
	for _, sub := range client.TimeSubscriptions {
		if sub.Instance == instance && sub.Processor == processor {
			return sub, true
		}
	}
	return nil, false
}

func (client *YamcsClient) HasTimeSubscriptionFor(instance string, processor string) bool {
	backend.Logger.Debug("checking time sub existence for", "instance", instance, "processor", processor)
	if sub, found := client.GetTimeSubscription(instance, processor); found {
		backend.Logger.Warn("found already existing time sub.", "instance", sub.Instance, "processor", sub.Processor)
		return true
	}
	return false
}
