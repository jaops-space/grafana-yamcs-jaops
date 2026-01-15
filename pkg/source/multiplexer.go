package multiplexer

import (
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// Multiplexer manages live parameter subscriptions, ensuring that only one subscription is active per parameter.
// It will automatically terminate subscriptions when they are no longer needed.
// It delegates connection management to ConnectionManager.
type Multiplexer struct {
	ConnMgr       *ConnectionManager
	Endpoints     map[string]*YamcsEndpoint
	Configuration *config.YamcsPluginConfiguration
	Secure        *config.YamcsSecureConfiguration

	SyncMux sync.Mutex
}

// NewMultiplexer creates a fresh multiplexer with a connection manager.
func NewMultiplexer(cfg *config.YamcsPluginConfiguration) *Multiplexer {
	return &Multiplexer{
		ConnMgr:       NewConnectionManager(cfg),
		Endpoints:     make(map[string]*YamcsEndpoint),
		Configuration: cfg,
		SyncMux:       sync.Mutex{},
	}
}

// SetupHost sets up a host for live subscriptions.
func (mux *Multiplexer) SetupHost(hostID string) error {
	return mux.ConnMgr.SetupHost(hostID)
}

// GetEndpoint retrieves or creates an Endpoint for the given ID.
 func (mux *Multiplexer) GetEndpoint(endpointID string) (*YamcsEndpoint, error) {
 	mux.SyncMux.Lock()
 	defer mux.SyncMux.Unlock()

 	endpointConfig, exists := mux.Configuration.Endpoints[endpointID]
 	if !exists {
 		return nil, exception.New("Configuration for endpoint "+endpointID+" not found", "ENDPOINT_CONFIG_NOT_FOUND")
 	}

 	// Get the Yamcs client from the connection manager
 	yamcsClient, err := mux.ConnMgr.GetClient(endpointConfig.Host)
 	if err != nil {
 		return nil, err
 	}

 	if endpoint, exists := mux.Endpoints[endpointID]; exists {
 		return endpoint, nil
 	}

 	instance, err := yamcsClient.GetInstanceByName(endpointConfig.Instance)
 	if err != nil {
 		return nil, err
 	}

 	processor, err := yamcsClient.GetProcessor(instance, endpointConfig.Processor)
 	if err != nil {
 		processor = yamcsClient.GetInstanceDefaultProcessor(instance)
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

 	// subscribe once per (instance, processor)
 	subscriptionExists := false
 	for _, subscription := range yamcsClient.ParameterSubscriptions {
 		if subscription.Instance == endpointConfig.Instance && subscription.Processor == endpointConfig.Processor {
 			subscriptionExists = true
 			break
 		}
 	}
 	if !subscriptionExists {
 		subscription, err := yamcsClient.CreateParameterSubscription(endpoint.Instance, endpoint.Processor)
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

// GetCommandHistoryListener returns a function that listens for command history entries.
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

// Dispose closes all websocket connections.
func (mux *Multiplexer) Dispose() {
	mux.ConnMgr.Dispose()
	mux.Endpoints = make(map[string]*YamcsEndpoint)
}