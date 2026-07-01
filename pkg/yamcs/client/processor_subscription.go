package client

import (
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
	subscriptionID int
	listener       ProcessorListener
	Instance       string
	Processor      string
	client         *YamcsClient
}

// CreateProcessorSubscription creates a new processor subscription.
func (client *YamcsClient) CreateProcessorSubscription(instance Instance, processor Processor) (*ProcessorSubscription, error) {
	subscription, err := client.newProcessorSubscription(instance.GetName(), processor.GetName())
	if err != nil {
		return nil, err
	}

	client.ProcessorSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// CreateProcessorSubscriptionByNames creates a processor subscription using plain names.
func (client *YamcsClient) CreateProcessorSubscriptionByNames(instance string, processor string) (*ProcessorSubscription, error) {
	subscription, err := client.newProcessorSubscription(instance, processor)
	if err != nil {
		return nil, err
	}

	client.ProcessorSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

func (client *YamcsClient) newProcessorSubscription(instance string, processor string) (*ProcessorSubscription, error) {
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

	_, callID, _, err := client.WebSocket.SendSync(message)
	if err != nil {
		return nil, err
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// HandleProcessorMessage processes incoming websocket messages for processor updates.
func (client *YamcsClient) HandleProcessorMessage(message *api.ServerMessage) {
	if message.GetType() != "processors" {
		return
	}

	processor := &yamcsManagement.ProcessorInfo{}
	if err := message.Data.UnmarshalTo(processor); err != nil {
		backend.Logger.Debug("Error unmarshalling processor data", "error", err)
		return
	}

	callID := message.GetCall()
	subscription, found := client.ProcessorSubscriptions[int(callID)]
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
	delete(subscription.client.ProcessorSubscriptions, subscription.subscriptionID)

	cancelRequest := &api.CancelOptions{
		Call: int32(subscription.subscriptionID),
	}

	anyMessage, _ := anypb.New(cancelRequest)

	message := &api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	}

	subscription.client.WebSocket.SendSync(message)
}
