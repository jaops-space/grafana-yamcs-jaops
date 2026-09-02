package source

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// GetLinksListener returns a function that listens for links updates from a specific Yamcs instance.
func (ep *YamcsEndpoint) getLinksListener() client.LinkListener {
	return func(event *links.LinkEvent) error {
		// Pushed exactly once, regardless of how many stream paths are
		// watching links on this endpoint - see getEventListener.
		ep.LinksRing.Push(event)

		ep.mu.RLock()
		defer ep.mu.RUnlock()
		for _, demand := range ep.LinkSignals {
			select {
			case demand.notify <- struct{}{}:
			default:
			}
		}
		return nil
	}
}

func (ep *YamcsEndpoint) RequestLinksStream(ctx context.Context, path string) error {

	_, err := ep.getOrCreateLinksSubscription(ctx)
	if err != nil {
		return err
	}
	ep.mu.Lock()
	ep.LinkSignals[path] = newBroadcastStreamDemand(ep.LinksRing)
	ep.mu.Unlock()

	return nil
}

// DrainLinksStream drains every link event pushed to the shared ring since
// this path's own cursor, and advances the cursor accordingly. Lock-free:
// the cursor is only ever touched by the single RunLinksStream goroutine
// that owns this path.
func (ep *YamcsEndpoint) DrainLinksStream(path string) []*links.LinkEvent {
	ep.mu.RLock()
	demand := ep.LinkSignals[path]
	ep.mu.RUnlock()
	if demand == nil {
		return nil
	}

	values, newCursor, dropped := ep.LinksRing.DrainSince(demand.cursor)
	demand.cursor = newCursor
	if dropped {
		backend.Logger.Warn("links stream fell behind its ring buffer capacity; oldest pending events were dropped", "path", path, "ringCapacity", BroadcastRingCapacity)
	}
	return values
}

// getOrCreateLinksSubscription does not touch mu - subscription
// lookup/creation is guarded by the client's own subsMu (for the map) and
// linkSubscribeMu (for the create race), so a slow/stuck subscribe attempt
// never blocks unrelated endpoint state guarded by mu.
func (ep *YamcsEndpoint) getOrCreateLinksSubscription(ctx context.Context) (*client.LinkSubscription, error) {

	cli, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := cli.FindLinkSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}

	ep.linkSubscribeMu.Lock()
	defer ep.linkSubscribeMu.Unlock()

	if subscription, found := cli.FindLinkSubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}
	subscription, err := cli.CreateLinkSubscription(ctx, ep.GetInstanceName(), ep.getLinksListener())
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

// GetLinksSignal returns the coalesced "new data available" notify channel
// for the given path, for RunLinksStream to select on.
func (ep *YamcsEndpoint) GetLinksSignal(path string) <-chan struct{} {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	demand := ep.LinkSignals[path]
	if demand == nil {
		return nil
	}
	return demand.notify
}

func (ep *YamcsEndpoint) WithdrawLinksStreamRequest(path string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if demand, ok := ep.LinkSignals[path]; ok {
		close(demand.notify)
		delete(ep.LinkSignals, path)
	}
	if len(ep.LinkSignals) == 0 {
		cli, err := ep.GetClient()
		if err != nil {
			return err
		}
		cli.HaltLinkSubscriptionsForInstance(ep.GetInstanceName())
	}
	return nil
}
