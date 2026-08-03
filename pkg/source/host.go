package source

import (
	"context"
	"sync"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

type YamcsHostInstance struct {
	Instance   client.Instance
	Processors map[string]client.Processor
}

// YamcsHost represents a Yamcs server connection along with its instances and processors.
type YamcsHost struct {
	mu            sync.RWMutex
	Client        *client.YamcsClient
	Instances     map[string]*YamcsHostInstance
	Configuration *config.YamcsHostConfiguration
}

func (host *YamcsHost) Name() string {
	return host.Configuration.DisplayName()
}

// retreive Host client
func (host *YamcsHost) GetClient() *client.YamcsClient {

	return host.Client

}

// SetupHost sets up a host for live subscriptions.
func (host *YamcsHost) Connect(ctx context.Context) error {

	client := host.GetClient()

	if client == nil {
		return exception.New("client not found", "HOST_CONNECT_WITHOUT_CLIENT")
	}

	if err := client.EstablishWebSocketConnection(ctx); err != nil {
		return err
	}

	return nil
}

func (host *YamcsHost) IsConnected() bool {

	client := host.GetClient()

	if client == nil {
		return false
	}

	return client.IsWebSocketConnected()

}

func (mux *Multiplexer) GetSecureData(host string) *config.YamcsSecureHost {
	if host == "" {
		return nil
	}
	secureHost, exists := mux.Secure.Hosts[host]
	if !exists {
		return nil
	}
	return secureHost
}

// GetProcessorListener updates processor snapshots and keeps endpoint processor references current.
func (host *YamcsHost) GetProcessorListener(instance client.Instance, processor client.Processor) func(update client.Processor) {

	instanceName := instance.GetName()
	processorName := processor.GetName()

	return func(update client.Processor) {

		if update == nil {
			return
		}

		host.mu.Lock()
		defer host.mu.Unlock()

		// TODO: error checking
		host.Instances[instanceName].Processors[processorName] = processor

	}
}
