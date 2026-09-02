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
		// Pushed exactly once, regardless of how many stream paths are
		// watching events on this endpoint - each demand's own cursor
		// determines which pushed values it has and hasn't drained yet.
		endpoint.EventsRing.Push(event)

		endpoint.mu.RLock()
		defer endpoint.mu.RUnlock()
		for _, demand := range endpoint.Events {
			select {
			case demand.notify <- struct{}{}:
			default:
			}
		}
	}
}

// RequestEventsStream initiates an event stream subscription.
func (ep *YamcsEndpoint) RequestEventsStream(ctx context.Context, path string) (<-chan struct{}, error) {

	_, err := ep.getOrCreateEventsSubscription(ctx)
	if err != nil {
		return nil, err
	}
	ep.mu.Lock()
	demand := newBroadcastStreamDemand(ep.EventsRing)
	ep.Events[path] = demand
	ep.mu.Unlock()

	return demand.notify, nil

}

// DrainEventsStream drains every event pushed to the shared ring since this
// path's own cursor, and advances the cursor accordingly. Lock-free: the
// cursor is only ever touched by the single RunEventStream goroutine that
// owns this path.
func (ep *YamcsEndpoint) DrainEventsStream(path string) []*events.Event {
	ep.mu.RLock()
	demand := ep.Events[path]
	ep.mu.RUnlock()
	if demand == nil {
		return nil
	}

	values, newCursor, dropped := ep.EventsRing.DrainSince(demand.cursor)
	demand.cursor = newCursor
	if dropped {
		backend.Logger.Warn("events stream fell behind its ring buffer capacity; oldest pending events were dropped", "path", path, "ringCapacity", BroadcastRingCapacity)
	}
	return values
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
	if demand, ok := ep.Events[path]; ok {
		close(demand.notify)
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
