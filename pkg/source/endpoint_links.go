package source

import (
	"context"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

type LinkSignal = chan *links.LinkEvent

// GetLinksListener returns a function that listens for links updates from a specific Yamcs instance.
func (ep *YamcsEndpoint) getLinksListener() client.LinkListener {
	return func(event *links.LinkEvent) error {
		ep.mu.RLock()
		defer ep.mu.RUnlock()
		for _, sig := range ep.LinkSignals {
			select {
			case sig <- event:
			default:
			}
		}
		return nil
	}
}

func (ep *YamcsEndpoint) RequestLinksStream(ctx context.Context, path string) error {

	ep.mu.RLock()
	_, err := ep.getOrCreateLinksSubscription(ctx)
	ep.mu.RUnlock()

	if err != nil {
		return err
	}
	ep.mu.Lock()
	ep.LinkSignals[path] = make(LinkSignal)
	ep.mu.Unlock()

	return err
}

func (ep *YamcsEndpoint) getOrCreateLinksSubscription(ctx context.Context) (*client.LinkSubscription, error) {

	cli, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	for _, subscription := range cli.LinkSubscriptions {
		if subscription.Instance == ep.GetInstanceName() {
			return subscription, nil
		}
	}
	subscription, err := cli.CreateLinkSubscription(ctx, ep.GetInstanceName())
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.getLinksListener())
	return subscription, nil
}

func (ep *YamcsEndpoint) GetLinksSignal(path string) LinkSignal {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.LinkSignals[path]
}

func (ep *YamcsEndpoint) WithdrawLinksStreamRequest(path string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	if signal, ok := ep.LinkSignals[path]; ok {
		close(signal)
		delete(ep.LinkSignals, path)
	}
	if len(ep.LinkSignals) == 0 {
		cli, err := ep.GetClient()
		if err != nil {
			return err
		}
		for _, subscription := range cli.LinkSubscriptions {
			if subscription.Instance == ep.GetInstanceName() {
				subscription.Halt()
			}
		}
	}
	return nil
}
