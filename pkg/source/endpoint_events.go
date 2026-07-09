package source

import (
	"context"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// GetEventListener returns a function that listens for events from a specific Yamcs instance.
func (endpoint *YamcsEndpoint) getEventListener() func(event *events.Event) {
	return func(event *events.Event) {
		endpoint.mu.RLock()
		defer endpoint.mu.RUnlock()
		for _, channel := range endpoint.Events {
			select {
			case channel <- event:
			default:
			}
		}
	}
}

// RequestEventsStream initiates an event stream subscription.
func (ep *YamcsEndpoint) RequestEventsStream(ctx context.Context, path string) (<-chan *events.Event, error) {

	ep.mu.RLock()
	_, err := ep.getOrCreateEventsSubscription(ctx)
	ep.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	ep.mu.Lock()
	ep.Events[path] = make(chan *events.Event)
	ep.mu.Unlock()

	return ep.Events[path], nil

}

func (ep *YamcsEndpoint) getOrCreateEventsSubscription(ctx context.Context) (*client.EventSubscription, error) {

	client, err := ep.GetClient()

	for _, subscription := range client.EventSubscriptions {
		if subscription.Instance == ep.GetInstanceName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateEventSubscription(ctx, ep.GetInstanceName())
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.getEventListener())

	return subscription, nil

}

// WithdrawEventsStreamRequest stops an event stream subscription.
func (ep *YamcsEndpoint) WithdrawEventsStreamRequest(path string) error {

	ep.mu.Lock()
	defer ep.mu.Unlock()
	if signal, ok := ep.Events[path]; ok {
		close(signal)
		delete(ep.Events, path)
	}

	if len(ep.Events) == 0 {
		client, err := ep.GetClient()
		if err != nil {
			return err
		}
		for _, subscription := range client.EventSubscriptions {
			if subscription.Instance == ep.GetInstanceName() {
				subscription.Halt()
			}
		}
	}
	return nil
}
