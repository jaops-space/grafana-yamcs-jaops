package client

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"google.golang.org/protobuf/types/known/anypb"
)

// LinkListener defines a callback for incoming links updates.
type LinkListener func(event *links.LinkEvent) error

// LinkSubscription manages a subscription to link status updates.
type LinkSubscription struct {
	subscriptionID int
	listener       LinkListener
	Instance       string
	client         *YamcsClient
}

// CreateLinkSubscription creates a new links subscription.
func (client *YamcsClient) CreateLinkSubscription(ctx context.Context, instance string) (*LinkSubscription, error) {
	subscription, err := client.newLinkSubscription(ctx, instance)
	if err != nil {
		return nil, err
	}

	client.LinkSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// newLinkSubscription initializes and subscribes to links updates.
func (client *YamcsClient) newLinkSubscription(ctx context.Context, instance string) (*LinkSubscription, error) {
	subscription := &LinkSubscription{
		client:   client,
		Instance: instance,
	}

	subscribeRequest := &links.SubscribeLinksRequest{
		Instance: new(instance),
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	message := &api.ClientMessage{
		Type:    "links",
		Options: anyMessage,
	}

	_, callID, _, err := client.WebSocket.SendSync(ctx, message)
	if err != nil {
		return nil, err
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// HandleLinkMessage processes incoming websocket messages for links updates.
func (client *YamcsClient) HandleLinkMessage(message *api.ServerMessage) {
	if message.GetType() != "links" {
		return
	}

	event := &links.LinkEvent{}
	if err := message.Data.UnmarshalTo(event); err != nil {
		backend.Logger.Debug("Error unmarshalling links event data", "error", err)
		return
	}

	callID := message.GetCall()
	subscription, found := client.LinkSubscriptions[int(callID)]
	if found && subscription.listener != nil {
		subscription.listener(event)
	}
}

// SetListener assigns a links listener to the subscription.
func (subscription *LinkSubscription) SetListener(listener LinkListener) {
	subscription.listener = listener
}

// Halt cancels the links subscription.
func (subscription *LinkSubscription) Halt() {
	delete(subscription.client.LinkSubscriptions, subscription.subscriptionID)

	cancelRequest := &api.CancelOptions{
		Call: int32(subscription.subscriptionID),
	}

	anyMessage, _ := anypb.New(cancelRequest)

	message := &api.ClientMessage{
		Type:    "cancel",
		Options: anyMessage,
	}

	subscription.client.WebSocket.Send(message)
}
