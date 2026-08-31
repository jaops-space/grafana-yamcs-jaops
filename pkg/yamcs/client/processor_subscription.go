package client

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/processing"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/yamcsManagement"
	"google.golang.org/protobuf/types/known/anypb"
)

// ProcessorListener defines a callback for incoming processor updates.
type ProcessorListener func(processor Processor)

// ProcessorSubscription manages a subscription to processor updates.
type ProcessorSubscription struct {
	subscriptionID int32
	listener       ProcessorListener
	Instance       string
	Processor      string
	client         *YamcsClient
}

// CreateProcessorSubscription creates a new processor subscription.
func (client *YamcsClient) CreateProcessorSubscription(ctx context.Context, instance Instance, processor Processor) (*ProcessorSubscription, error) {
	subscription, err := client.newProcessorSubscription(ctx, instance.GetName(), processor.GetName())
	if err != nil {
		return nil, err
	}

	client.subsMu.Lock()
	client.ProcessorSubscriptions[subscription.subscriptionID] = subscription
	client.subsMu.Unlock()
	return subscription, nil
}

// CreateProcessorSubscriptionByNames creates a processor subscription using plain names.
func (client *YamcsClient) CreateProcessorSubscriptionByNames(ctx context.Context, instance string, processor string) (*ProcessorSubscription, error) {
	subscription, err := client.newProcessorSubscription(ctx, instance, processor)
	if err != nil {
		return nil, err
	}

	client.subsMu.Lock()
	client.ProcessorSubscriptions[subscription.subscriptionID] = subscription
	client.subsMu.Unlock()
	return subscription, nil
}

func (client *YamcsClient) newProcessorSubscription(ctx context.Context, instance string, processor string) (*ProcessorSubscription, error) {
	cooldownKey := subscribeCooldownKey("processors", instance, processor)
	if err := client.checkSubscribeCooldown(cooldownKey); err != nil {
		return nil, err
	}

	subscription := &ProcessorSubscription{
		client:    client,
		Instance:  instance,
		Processor: processor,
	}

	subscribeRequest := &processing.SubscribeProcessorsRequest{
		Instance:  &instance,
		Processor: &processor,
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "processors",
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

// HandleProcessorMessage processes incoming websocket messages for processor updates.
func (client *YamcsClient) HandleProcessorMessage(message *api.ServerMessage) {

	processor := &yamcsManagement.ProcessorInfo{}
	if err := message.Data.UnmarshalTo(processor); err != nil {
		backend.Logger.Debug("Error unmarshalling processor data", "error", err)
		return
	}

	callID := message.GetCall()
	client.subsMu.RLock()
	subscription, found := client.ProcessorSubscriptions[callID]
	client.subsMu.RUnlock()
	if found && subscription.listener != nil {
		subscription.listener(processor)
	}
}

// SetListener assigns a processor listener to the subscription.
func (subscription *ProcessorSubscription) SetListener(listener ProcessorListener) {
	subscription.listener = listener
}

// Halt cancels the processor subscription.
func (subscription *ProcessorSubscription) Halt() {
	subscription.client.subsMu.Lock()
	delete(subscription.client.ProcessorSubscriptions, subscription.subscriptionID)
	subscription.client.subsMu.Unlock()

	cancelRequest := &api.CancelOptions{
		Call: subscription.subscriptionID,
	}

	anyMessage, _ := anypb.New(cancelRequest)

	message := &api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	}

	subscription.client.WebSocket.Send(message)
}
