package client

import (
	"fmt"
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
	subscriptionID int
	Instance       string
	Processor      string
	listener       TimeListener
	client         *YamcsClient
}

func (client *YamcsClient) CreateTimeSubscription(instance Instance, processor Processor) (*TimeSubscription, error) {

	subscription, err := NewTimeSubscription(client, instance.GetName(), processor.GetName())
	if err != nil {
		return nil, err
	}

	client.TimeSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil

}

// SubscribeTime subscribes to time updates from a specific instance and processor.
func NewTimeSubscription(client *YamcsClient, instance string, processor string) (*TimeSubscription, error) {

	// Create the subscription request for time updates
	subscribeTimeRequest := &ptime.SubscribeTimeRequest{
		Instance:  &instance,
		Processor: &processor,
	}

	// Convert the subscription request into an Any message
	anyMessage, err := anypb.New(subscribeTimeRequest)
	if err != nil {
		fmt.Printf("Error creating Any message. Details: %v", err)
		return nil, err
	}

	// Prepare the message to send via WebSocket
	message := &api.ClientMessage{
		Type:    "time",     // Message type indicating it's a time subscription
		Id:      32,         // Unique message identifier
		Options: anyMessage, // Attach the Any message containing the subscription request
	}

	_, callID, _, err := client.WebSocket.SendSync(message)
	if err != nil {
		return nil, err
	}

	subscription := &TimeSubscription{
		subscriptionID: callID,
		Instance:       instance,
		Processor:      processor,
		client:         client,
	}

	backend.Logger.Info("subscribing to procssor time", "proc", processor)

	return subscription, nil
}

func (subscription *TimeSubscription) Halt() {

	delete(subscription.client.TimeSubscriptions, subscription.subscriptionID)

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

func (client *YamcsClient) HandleTimeMessage(message *api.ServerMessage) {

	if message.GetType() == "time" {
		timestamp := &timestamppb.Timestamp{}
		if err := message.Data.UnmarshalTo(timestamp); err != nil {
			panic(exception.Wrap("Unmarshal error", "SUBSCRIPTION_UNMARSHALL_ERROR", err))
		}

		// Retrieve the subscription by call ID
		callID := message.GetCall()
		subscription, found := client.TimeSubscriptions[int(callID)]
		if !found {
			return
		}

		if subscription.listener != nil {
			backend.Logger.Debug("received time", "time", timestamp.AsTime(), "callID", callID)
			subscription.listener(timestamp.AsTime())
		}
	}

}

func (subscription *TimeSubscription) SetTimeListener(listener TimeListener) {
	subscription.listener = listener
}

func (client *YamcsClient) HasTimeSubscriptionFor(instance Instance, processor Processor) bool {
	backend.Logger.Info("checking time sub existence for", "instance", instance.GetName(), "processor", processor.GetName())
	for _, sub := range client.TimeSubscriptions {
		if sub.Instance == instance.GetName() && sub.Processor == processor.GetName() {
			backend.Logger.Warn("found already existing time sub.", "instance", sub.Instance, "processor", sub.Processor)
			return true
		}
	}
	return false
}
