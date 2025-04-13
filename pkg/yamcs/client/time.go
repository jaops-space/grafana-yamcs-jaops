package client

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/time"
	"google.golang.org/protobuf/types/known/anypb"
)

// SubscribeTime subscribes to time updates from a specific instance and processor.
func (client *YamcsClient) SubscribeTime(instance Instance, processor Processor) {

	// Create the subscription request for time updates
	subscribeTimeRequest := &time.SubscribeTimeRequest{
		Instance:  instance.Name,
		Processor: processor.Name,
	}

	// Convert the subscription request into an Any message
	anyMessage, err := anypb.New(subscribeTimeRequest)
	if err != nil {
		fmt.Printf("Error creating Any message. Details: %v", err)
		return
	}

	// Prepare the message to send via WebSocket
	message := &api.ClientMessage{
		Type:    "time",     // Message type indicating it's a time subscription
		Id:      32,         // Unique message identifier
		Options: anyMessage, // Attach the Any message containing the subscription request
	}

	// Send the message via WebSocket
	client.WebSocket.Send(message)
}
