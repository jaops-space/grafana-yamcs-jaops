package multiplexer

import (
	"fmt"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// Multiplexer manages parameter requests, ensuring that only one subscription is active per parameter.
// It will automatically terminate subscriptions when they are no longer needed.
type Multiplexer struct {
	Hosts         map[string]*YamcsHost
	Endpoints     map[string]*YamcsEndpoint
	Configuration *config.YamcsPluginConfiguration
	Secure        *config.YamcsSecureConfiguration
	SyncMux       sync.Mutex
}

// YamcsHost represents a Yamcs server connection along with its instances and processors.
type YamcsHost struct {
	Client     *client.YamcsClient
	Instances  map[string]client.Instance
	Processors map[string]client.Processor
}

// NewMultiplexer initializes a new Multiplexer instance with the provided configuration.
func NewMultiplexer(cfg *config.YamcsPluginConfiguration) *Multiplexer {
	return &Multiplexer{
		Hosts:         make(map[string]*YamcsHost),
		Endpoints:     make(map[string]*YamcsEndpoint),
		Configuration: cfg,
		SyncMux:       sync.Mutex{},
	}
}

// SetupHost configures and connects a Yamcs host for parameter streaming.
func (mux *Multiplexer) SetupHost(hostID string) error {

	hostConfig, exists := mux.Configuration.Hosts[hostID]
	if !exists {
		return exception.New(fmt.Sprintf("Configuration for host %s not found", hostID), "CONFIGURATION_NOT_FOUND")
	}

	var tlsConfig http.TLS
	var creds http.Credentials

	if hostConfig.Tls {
		tlsConfig = http.GetTLSConfiguration(!hostConfig.TlsInsecure)
	} else {
		tlsConfig = http.GetNoTLSConfiguration()
	}

	if !hostConfig.Auth {
		creds = &http.NoCredentials{}
	} else {
		username := hostConfig.Username
		secure := mux.GetSecureData(hostID)
		if secure == nil {
			return exception.New(fmt.Sprintf("Secure configuration for host %s not found", hostID), "SECURE_CONFIGURATION_NOT_FOUND")
		}
		password := secure.Password
		creds = &http.BasicAuthCredentials{
			Username: username,
			Password: password,
		}
	}

	yamcsClient, err := client.NewYamcsClient(hostConfig.Path, tlsConfig, creds)
	if err != nil {
		return err
	}

	if err = yamcsClient.EstablishWebSocketConnection(); err != nil {
		return err
	}

	mux.Hosts[hostID] = &YamcsHost{
		Client:     yamcsClient,
		Instances:  make(map[string]client.Instance),
		Processors: make(map[string]client.Processor),
	}

	return nil
}

// GetEndpoint retrieves or creates an Endpoint for the given ID.
func (mux *Multiplexer) GetEndpoint(endpointID string) (*YamcsEndpoint, error) {

	mux.SyncMux.Lock()
	defer mux.SyncMux.Unlock()

	endpointConfig, exists := mux.Configuration.Endpoints[endpointID]
	if !exists {
		return nil, exception.New("Configuration for endpoint "+endpointID+" not found", "ENDPOINT_CONFIG_NOT_FOUND")
	}

	host, exists := mux.Hosts[endpointConfig.Host]
	if !exists {
		if err := mux.SetupHost(endpointConfig.Host); err != nil {
			return nil, err
		}
		host = mux.Hosts[endpointConfig.Host]
	}

	if host.Client == nil {
		return nil, exception.New("Unexpected error retrieving Yamcs client", "MULTIPLEXER_CLIENT_NOT_FOUND")
	}

	if !host.Client.IsWebSocketConnected() {
		if err := host.Client.EstablishWebSocketConnection(); err != nil {
			return nil, err
		}
	}

	if endpoint, exists := mux.Endpoints[endpointID]; exists {
		return endpoint, nil
	}

	instance, err := host.Client.GetInstanceByName(endpointConfig.Instance)
	if err != nil {
		return nil, err
	}

	processor, err := host.Client.GetProcessor(instance, endpointConfig.Processor)
	if err != nil {
		processor = host.Client.GetInstanceDefaultProcessor(instance)
		if processor == nil {
			return nil, err
		}
	}

	endpoint := &YamcsEndpoint{
		Multiplexer:    mux,
		Parameters:     make(map[string]*ParameterDemand),
		Events:         make(map[string][]*events.Event),
		CommandHistory: make(map[string][]*commanding.CommandHistoryEntry),
		ID:             endpointID,
		Instance:       instance,
		Processor:      processor,
	}
	mux.Endpoints[endpointID] = endpoint

	subscriptionExists := false
	for _, subscription := range host.Client.ParameterSubscriptions {
		if subscription.Instance == endpointConfig.Instance && subscription.Processor == endpointConfig.Processor {
			subscriptionExists = true
			break
		}
	}

	if !subscriptionExists {
		subscription, err := host.Client.CreateParameterSubscription(endpoint.Instance, endpoint.Processor)
		if err != nil {
			return nil, err
		}
		subscription.SetListener(endpoint.GetChannelParameterListener())
	}

	backend.Logger.Info("created endpoint", "endpoint", endpoint, "current endpoints", mux.Endpoints)

	return endpoint, nil
}

// GetEventListener returns a function that listens for events from a specific Yamcs instance.
func (mux *Multiplexer) GetEventListener(instance client.Instance) func(event *events.Event) {
	return func(event *events.Event) {
		for _, dataSource := range mux.Endpoints {
			if dataSource.Instance.GetName() == instance.GetName() {
				for path := range dataSource.Events {
					dataSource.Events[path] = append(dataSource.Events[path], event)
				}
			}
		}
	}
}

// GetCommandHistoryListener returns a function that listens for command history entries from a specific Yamcs instance.
func (mux *Multiplexer) GetCommandHistoryListener(instance client.Instance) func(entry *commanding.CommandHistoryEntry) {
	return func(entry *commanding.CommandHistoryEntry) {
		for _, dataSource := range mux.Endpoints {
			if dataSource.Instance.GetName() == instance.GetName() {
				for path := range dataSource.CommandHistory {
					dataSource.CommandHistory[path] = append(dataSource.CommandHistory[path], entry)
				}
			}
		}
	}
}

func (mux *Multiplexer) Dispose() {
	// Close all host clients (this closes WebSocket connections)
	for _, host := range mux.Hosts {
		if host.Client != nil {
			host.Client.CloseWebSocketConnection()
		}
	}

	// Close any remaining endpoint connections
	for _, endpoint := range mux.Endpoints {
		client := endpoint.GetClient()
		if client != nil {
			client.CloseWebSocketConnection()
		}
	}

	// Clear all cached state
	mux.Endpoints = make(map[string]*YamcsEndpoint)
	mux.Hosts = make(map[string]*YamcsHost)
}
