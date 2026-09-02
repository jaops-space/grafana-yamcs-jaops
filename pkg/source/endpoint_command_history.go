package source

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// GetCommandHistoryListener returns a function that listens for command history entries.
func (ep *YamcsEndpoint) getCommandHistoryListener() client.CommandHistoryListener {
	return func(entry *commanding.CommandHistoryEntry) error {
		// Pushed exactly once, regardless of how many stream paths are
		// watching command history on this endpoint - see getEventListener.
		ep.CommandHistoryRing.Push(entry)

		ep.mu.RLock()
		defer ep.mu.RUnlock()
		for _, demand := range ep.CommandHistorySignals {
			select {
			case demand.notify <- struct{}{}:
			default:
			}
		}
		return nil
	}
}

func (ep *YamcsEndpoint) RequestCommandHistoryStream(ctx context.Context, path string) error {

	_, err := ep.getOrCreateCommandHistorySubscription(ctx)
	if err != nil {
		return err
	}
	ep.mu.Lock()
	ep.CommandHistorySignals[path] = newBroadcastStreamDemand(ep.CommandHistoryRing)
	ep.mu.Unlock()
	return nil

}

// DrainCommandHistoryStream drains every entry pushed to the shared ring
// since this path's own cursor, and advances the cursor accordingly.
// Lock-free: the cursor is only ever touched by the single
// RunCommandHistoryStream goroutine that owns this path.
func (ep *YamcsEndpoint) DrainCommandHistoryStream(path string) []*commanding.CommandHistoryEntry {
	ep.mu.RLock()
	demand := ep.CommandHistorySignals[path]
	ep.mu.RUnlock()
	if demand == nil {
		return nil
	}

	values, newCursor, dropped := ep.CommandHistoryRing.DrainSince(demand.cursor)
	demand.cursor = newCursor
	if dropped {
		backend.Logger.Warn("command history stream fell behind its ring buffer capacity; oldest pending entries were dropped", "path", path, "ringCapacity", BroadcastRingCapacity)
	}
	return values
}

// getOrCreateCommandHistorySubscription does not touch mu - subscription
// lookup/creation is guarded by the client's own subsMu (for the map) and
// commandHistorySubMu (for the create race), so a slow/stuck subscribe
// attempt never blocks unrelated endpoint state guarded by mu.
func (ep *YamcsEndpoint) getOrCreateCommandHistorySubscription(ctx context.Context) (*client.CommandHistorySubscription, error) {

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := client.FindCommandHistorySubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}

	ep.commandHistorySubMu.Lock()
	defer ep.commandHistorySubMu.Unlock()

	if subscription, found := client.FindCommandHistorySubscription(ep.GetInstanceName()); found {
		return subscription, nil
	}
	subscription, err := client.CreateCommandHistorySubscription(ctx, ep.GetInstanceName(), ep.GetProcessorName(), ep.getCommandHistoryListener())
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

// GetCommandHistorySignal returns the coalesced "new data available" notify
// channel for the given path, for RunCommandHistoryStream to select on.
func (ep *YamcsEndpoint) GetCommandHistorySignal(path string) <-chan struct{} {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	demand := ep.CommandHistorySignals[path]
	if demand == nil {
		return nil
	}
	return demand.notify
}

func (ep *YamcsEndpoint) WithdrawCommandHistoryStreamRequest(path string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if demand, ok := ep.CommandHistorySignals[path]; ok {
		close(demand.notify)
		delete(ep.CommandHistorySignals, path)
	}
	if len(ep.CommandHistorySignals) == 0 {
		client, err := ep.GetClient()
		if err != nil {
			return err
		}
		client.HaltCommandHistorySubscriptionsForInstance(ep.GetInstanceName())
	}
	return nil
}
