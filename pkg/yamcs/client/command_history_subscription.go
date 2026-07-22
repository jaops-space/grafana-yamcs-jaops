package client

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// CommandHistoryListener defines a callback for incoming command history entries.
type CommandHistoryListener func(entry *commanding.CommandHistoryEntry) error

// CommandHistorySubscription manages a subscription to command history updates.
type CommandHistorySubscription struct {
	subscriptionID      int32
	activeSubscriptions types.Set[string]
	commandListener     CommandHistoryListener
	Instance            string
	client              *YamcsClient
}

// CreateCommandHistorySubscription creates a new command history subscription.
func (client *YamcsClient) CreateCommandHistorySubscription(instance string, processor string) (*CommandHistorySubscription, error) {
	subscription, err := client.newCommandHistorySubscription(instance, processor)
	if err != nil {
		return nil, err
	}

	client.CommandHistorySubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// newCommandHistorySubscription initializes and subscribes to command history.
func (client *YamcsClient) newCommandHistorySubscription(instance, processor string) (*CommandHistorySubscription, error) {
	subscription := &CommandHistorySubscription{
		client:              client,
		Instance:            instance,
		activeSubscriptions: types.Set[string]{},
	}

	// Prepare subscription request
	subscribeRequest := &commanding.SubscribeCommandsRequest{
		Instance:  &instance,
		Processor: &processor,
	}

	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	// Send the subscription request via WebSocket
	message := &api.ClientMessage{
		Type:    "commands",
		Options: anyMessage,
	}

	_, callID, _, err := client.WebSocket.SendSync(context.Background(), message)
	if err != nil {
		return nil, err
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// HandleCommandMessage processes incoming WebSocket messages for command history.
func (client *YamcsClient) HandleCommandMessage(message *api.ServerMessage) {
	if message.GetType() == "commands" {
		entry := &commanding.CommandHistoryEntry{}
		if err := message.Data.UnmarshalTo(entry); err != nil {
			backend.Logger.Debug("Error unmarshalling command history data", "error", err)
			return
		}

		callID := message.GetCall()
		subscription, found := client.CommandHistorySubscriptions[callID]
		if found && subscription.commandListener != nil {
			subscription.commandListener(entry)
		}
	}
}

// SetListener assigns a command history listener to the subscription.
func (subscription *CommandHistorySubscription) SetListener(listener CommandHistoryListener) {
	subscription.commandListener = listener
}

// Halt cancels the command history subscription.
func (subscription *CommandHistorySubscription) Halt() {

	delete(subscription.client.CommandHistorySubscriptions, subscription.subscriptionID)

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
