package source

import (
	"context"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

type CommandHistorySignal = chan *commanding.CommandHistoryEntry

// GetCommandHistoryListener returns a function that listens for command history entries.
func (ep *YamcsEndpoint) getCommandHistoryListener() client.CommandHistoryListener {
	return func(entry *commanding.CommandHistoryEntry) error {
		ep.mu.RLock()
		defer ep.mu.RUnlock()
		for _, sig := range ep.CommandHistorySignals {
			select {
			case sig <- entry:
			default:
			}
		}
		return nil
	}
}

func (ep *YamcsEndpoint) RequestCommandHistoryStream(ctx context.Context, path string) error {

	ep.mu.RLock()
	_, err := ep.getOrCreateCommandHistorySubscription(ctx)
	ep.mu.RUnlock()
	if err != nil {
		return err
	}
	ep.mu.Lock()
	ep.CommandHistorySignals[path] = make(chan *commanding.CommandHistoryEntry)
	ep.mu.Unlock()
	return nil

}

func (ep *YamcsEndpoint) getOrCreateCommandHistorySubscription(ctx context.Context) (*client.CommandHistorySubscription, error) {

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	for _, subscription := range client.CommandHistorySubscriptions {
		if subscription.Instance == ep.GetInstanceName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateCommandHistorySubscription(ctx, ep.GetInstanceName(), ep.GetProcessorName())
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.getCommandHistoryListener())
	return subscription, nil
}

func (ep *YamcsEndpoint) GetCommandHistorySignal(path string) CommandHistorySignal {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.CommandHistorySignals[path]
}

func (ep *YamcsEndpoint) WithdrawCommandHistoryStreamRequest(path string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if signal, ok := ep.CommandHistorySignals[path]; ok {
		close(signal)
		delete(ep.CommandHistorySignals, path)
	}
	if len(ep.CommandHistorySignals) == 0 {
		client, err := ep.GetClient()
		if err != nil {
			return err
		}
		for _, subscription := range client.CommandHistorySubscriptions {
			if subscription.Instance == ep.GetInstanceName() {
				subscription.Halt()
			}
		}
	}
	return nil
}
