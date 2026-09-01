package source

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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
				backend.Logger.Warn("dropping event because stream buffer is full", "message", event.GetMessage())
			}
		}
	}
}

// RequestEventsStream initiates an event stream subscription.
func (ep *YamcsEndpoint) RequestEventsStream(ctx context.Context, path string) (<-chan *events.Event, error) {

	_, err := ep.getOrCreateEventsSubscription(ctx)
	if err != nil {
		return nil, err
	}
	ep.mu.Lock()
	ep.Events[path] = make(chan *events.Event, StreamSignalBufferSize)
	signal := ep.Events[path]
	ep.mu.Unlock()

	return signal, nil

}

func (ep *YamcsEndpoint) DrainEventsSignal(first *events.Event, signal <-chan *events.Event) []*events.Event {
	drained := []*events.Event{first}
	for {
		select {
		case event, ok := <-signal:
			if !ok {
				return drained
			}
			drained = append(drained, event)
		default:
			return drained
		}
	}
}

// getOrCreateEventsSubscription does not touch mu - subscription
// lookup/creation is guarded by the client's own subsMu (for the map) and
// eventSubscribeMu (for the create race), so a slow/stuck subscribe attempt
// never blocks unrelated endpoint state guarded by mu.
func (ep *YamcsEndpoint) getOrCreateEventsSubscription(ctx context.Context) (*client.EventSubscription, error) {

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := client.FindEventSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}

	ep.eventSubscribeMu.Lock()
	defer ep.eventSubscribeMu.Unlock()

	if subscription, found := client.FindEventSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}
	subscription, err := client.CreateEventSubscription(ctx, ep.GetInstanceName(), ep.getEventListener())
	if err != nil {
		return nil, err
	}

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
		client.HaltEventSubscriptionsForInstance(ep.GetInstanceName())
	}
	return nil
}
