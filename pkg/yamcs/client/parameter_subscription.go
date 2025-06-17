package client

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/processing"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// ParameterListener is a callback function that is invoked when a parameter's value changes.
type ParameterListener func(parameter string, newValue *pvalue.ParameterValue)

// ParameterSubscription manages a subscription to a set of parameters from a Yamcs instance and processor.
type ParameterSubscription struct {
	subscriptionID      int
	parameterIDToName   map[int]string    // Maps internal parameter IDs to names
	ActiveSubscriptions types.Set[string] // Set of currently subscribed parameters
	valueChangeListener ParameterListener // Listener for value changes
	Instance            string
	Processor           string
	client              *YamcsClient
}

// NewParameterSubscription creates a new ParameterSubscription for an instance and processor with initial parameters.
func NewParameterSubscription(client *YamcsClient, instanceName, processorName string, initialParameters ...string) (*ParameterSubscription, error) {
	subscription := &ParameterSubscription{
		client:              client,
		Instance:            instanceName,
		Processor:           processorName,
		parameterIDToName:   make(map[int]string),
		ActiveSubscriptions: types.Set[string]{},
	}

	// Create subscription request
	subscribeRequest := &processing.SubscribeParametersRequest{
		Instance:  &instanceName,
		Processor: &processorName,
	}

	// Add parameters to subscription
	var namedObjectIds []*protobuf.NamedObjectId
	for _, param := range initialParameters {
		namedObjectIds = append(namedObjectIds, &protobuf.NamedObjectId{Name: &param})
	}
	subscribeRequest.Id = namedObjectIds

	// Marshal the request into a message
	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return nil, err
	}

	// Send subscription request over WebSocket
	message := &api.ClientMessage{
		Type:    "parameters",
		Options: anyMessage,
	}
	_, callID, _, err := client.WebSocket.SendSync(message)
	if err != nil {
		return nil, err
	}

	// Update the active subscriptions set
	for _, param := range initialParameters {
		subscription.ActiveSubscriptions.Add(param)
	}

	subscription.subscriptionID = callID
	return subscription, nil
}

// Add subscribes to additional parameters by their qualified names.
func (sub *ParameterSubscription) Add(parameters ...string) error {

	// Send the add request
	if err := sub.updateSubscription(processing.SubscribeParametersRequest_ADD, parameters...); err != nil {
		return err
	}

	// Add parameters to the active set
	for _, param := range parameters {
		sub.ActiveSubscriptions.Add(param)
	}

	return nil
}

// Remove unsubscribes from parameters by their qualified names.
func (sub *ParameterSubscription) Remove(parameters ...string) error {

	// Send the remove request
	if err := sub.updateSubscription(processing.SubscribeParametersRequest_REMOVE, parameters...); err != nil {
		return err
	}

	// Remove parameters from the active set
	for _, param := range parameters {
		sub.ActiveSubscriptions.Remove(param)
	}

	return nil
}

// Replace replaces the current subscriptions with new ones, unsubscribing from the old ones.
func (sub *ParameterSubscription) Replace(parameters ...Parameter) error {
	parameterNames := make([]string, len(parameters))
	for i, param := range parameters {
		parameterNames[i] = param.GetQualifiedName()
	}

	// Send the replace request
	if err := sub.updateSubscription(processing.SubscribeParametersRequest_REPLACE, parameterNames...); err != nil {
		return err
	}

	// Clear active subscriptions and add the new ones
	sub.ActiveSubscriptions = make(types.Set[string])
	for _, param := range parameters {
		sub.ActiveSubscriptions.Add(param.GetQualifiedName())
	}

	return nil
}

// Checks if subscription has a certain parameter
func (sub *ParameterSubscription) Has(parameter string) bool {
	return sub.ActiveSubscriptions.Exists(parameter)
}

// SetListener sets the listener function that is called when parameter values change.
func (sub *ParameterSubscription) SetListener(listener ParameterListener) {
	sub.valueChangeListener = listener
}

// updateSubscription sends an update to the server with the given action (add, remove, replace).
func (sub *ParameterSubscription) updateSubscription(action processing.SubscribeParametersRequest_Action, parameters ...string) error {
	// Create and populate the subscription request
	subscribeRequest := &processing.SubscribeParametersRequest{
		Instance:  &sub.Instance,
		Processor: &sub.Processor,
		Action:    action.Enum(),
	}

	var namedObjectIds []*protobuf.NamedObjectId
	for _, param := range parameters {
		namedObjectIds = append(namedObjectIds, &protobuf.NamedObjectId{Name: &param})
	}
	subscribeRequest.Id = namedObjectIds

	// Create the message and send it via WebSocket
	anyMessage, err := anypb.New(subscribeRequest)
	if err != nil {
		return err
	}

	message := &api.ClientMessage{
		Type:    "parameters",
		Call:    int32(sub.subscriptionID),
		Options: anyMessage,
	}

	return sub.client.WebSocket.Send(message)
}

// HandleParameterMessage handles incoming parameter updates from the server and invokes the listener.
func (client *YamcsClient) HandleParameterMessage(message *api.ServerMessage) {

	if message.GetType() == "parameters" {
		parameterData := &processing.SubscribeParametersData{}
		if err := message.Data.UnmarshalTo(parameterData); err != nil {
			panic(exception.Wrap("Unmarshal error", "SUBSCRIPTION_UNMARSHALL_ERROR", err))
		}

		// Retrieve the subscription by call ID
		callID := message.GetCall()
		subscription, found := client.ParameterSubscriptions[int(callID)]
		if !found {
			return
		}

		// Map parameter IDs to names
		if parameterData.Mapping != nil {
			for key, param := range parameterData.GetMapping() {
				subscription.parameterIDToName[int(key)] = param.GetName()
			}
			for _, invalidParam := range parameterData.GetInvalid() {
				backend.Logger.Warn("Invalid subscription parameter ID", "id", invalidParam)
			}
		}

		// Invoke the listener for each parameter value
		if subscription.valueChangeListener != nil {
			for _, value := range parameterData.GetValues() {
				paramName, found := subscription.parameterIDToName[int(value.GetNumericId())]
				if !found {
					backend.Logger.Warn("Unknown parameter ID", "id", value.GetNumericId())
					continue
				}
				subscription.valueChangeListener(paramName, value)
			}
		}
	}
}

// CreateParameterSubscription creates a new subscription for a set of parameters and adds it to the client's subscription registry.
func (client *YamcsClient) CreateParameterSubscription(instance Instance, processor Processor, initialParameters ...Parameter) (*ParameterSubscription, error) {
	parameterNames := make([]string, len(initialParameters))
	for i, param := range initialParameters {
		parameterNames[i] = param.GetQualifiedName()
	}

	subscription, err := NewParameterSubscription(client, instance.GetName(), processor.GetName(), parameterNames...)
	if err != nil {
		return nil, err
	}

	client.ParameterSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// CreateParameterSubscription creates a new subscription for a set of parameters and adds it to the client's subscription registry.
func (client *YamcsClient) CreateParameterSubscriptionByNames(instance Instance, processor Processor, initialParameters ...string) (*ParameterSubscription, error) {

	subscription, err := NewParameterSubscription(client, instance.GetName(), processor.GetName(), initialParameters...)
	if err != nil {
		return nil, err
	}

	client.ParameterSubscriptions[subscription.subscriptionID] = subscription
	return subscription, nil
}

// ClearParameterSubscriptions clears all active parameter subscriptions.
func (client *YamcsClient) ClearParameterSubscriptions() {
	for id := range client.ParameterSubscriptions {
		delete(client.ParameterSubscriptions, id)
	}
}

func (subscription *ParameterSubscription) Halt() {

	delete(subscription.client.ParameterSubscriptions, subscription.subscriptionID)

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
