package client

import (
	"log"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// CommandHistoryListener defines a callback for incoming command history entries.
type CommandHistoryListener func(entry *commanding.CommandHistoryEntry)

// CommandHistorySubscription manages a subscription to command history updates.
type CommandHistorySubscription struct {
	subscriptionID      int
	activeSubscriptions types.Set[string]
	commandListener     CommandHistoryListener
	Instance            string
	client              *YamcsClient
}

// CreateCommandHistorySubscription creates a new command history subscription.
func (client *YamcsClient) CreateCommandHistorySubscription(instance Instance, processor Processor) (*CommandHistorySubscription, error) {
	subscription, err := client.newCommandHistorySubscription(instance.GetName(), processor.GetName())
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

	_, callID, _, err := client.WebSocket.SendSync(message)
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
			log.Default().Printf("Error unmarshalling command history data: %v\n", err)
			return
		}

		callID := message.GetCall()
		subscription, found := client.CommandHistorySubscriptions[int(callID)]
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
