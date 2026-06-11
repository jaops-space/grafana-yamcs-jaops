package source

import (
	"fmt"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"google.golang.org/protobuf/proto"
)

// Multiplexer manages live parameter subscriptions, ensuring that only one subscription is active per parameter.
// It will automatically terminate subscriptions when they are no longer needed.
// It delegates connection management to ConnectionManager.
type Multiplexer struct {
	Hosts         map[string]*YamcsHost
	Endpoints     map[string]*YamcsEndpoint
	Configuration *config.YamcsPluginConfiguration
	Secure        *config.YamcsSecureConfiguration

	SyncMux sync.Mutex
}

// NewMultiplexer creates a fresh multiplexer with a connection manager.
func NewMultiplexer(cfg *config.YamcsPluginConfiguration) *Multiplexer {
	return &Multiplexer{
		Hosts:         make(map[string]*YamcsHost),
		Endpoints:     make(map[string]*YamcsEndpoint),
		Configuration: cfg,
		SyncMux:       sync.Mutex{},
	}
}

// GetEndpoint retrieves or creates an Endpoint for the given ID.
func (mux *Multiplexer) GetEndpoint(endpointID string) (*YamcsEndpoint, error) {
	mux.SyncMux.Lock()
	defer mux.SyncMux.Unlock()

	// add logs

	backend.Logger.Debug("retrieving endpoint", "endpointID", endpointID)
	endpointConfig, exists := mux.Configuration.Endpoints[endpointID]
	if !exists {
		return nil, exception.New("Configuration for endpoint "+endpointID+" not found", "ENDPOINT_CONFIG_NOT_FOUND")
	}

	// Get the Yamcs client from the connection manager
	backend.Logger.Debug("retrieving Yamcs client for host", "hostID", endpointConfig.Host)
	yamcsClient, err := mux.GetClient(endpointConfig.Host)
	if err != nil {
		return nil, err
	}

	if endpoint, exists := mux.Endpoints[endpointID]; exists {
		return endpoint, nil
	}

	backend.Logger.Debug("creating new endpoint", "endpointID", endpointID, "instance", endpointConfig.Instance, "processor", endpointConfig.Processor)
	instance, err := yamcsClient.GetInstanceByName(endpointConfig.Instance)
	if err != nil {
		return nil, err
	}

	backend.Logger.Debug("retrieving processor for instance", "instance", instance.GetName(), "processor", endpointConfig.Processor)
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
		Alarms:         make(map[string][]*alarms.AlarmData),
		AlarmCache:     make(map[string]*alarms.AlarmData),
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

	backend.Logger.Debug("created endpoint", "endpoint", endpoint, "current endpoints", mux.Endpoints)

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

// GetAlarmsListener returns a function that listens for alarm events from a specific Yamcs instance.
func (mux *Multiplexer) GetAlarmsListener(instance client.Instance) func(alarm *alarms.AlarmData) {
	return func(alarm *alarms.AlarmData) {
		for _, dataSource := range mux.Endpoints {
			if dataSource.Instance.GetName() == instance.GetName() {
				// Generate unique alarm ID (namespace/name/seqNum)
				qualifiedName := alarm.GetId().GetNamespace() + "/" + alarm.GetId().GetName()
				alarmID := fmt.Sprintf("%s/%d", qualifiedName, alarm.GetSeqNum())

				dataSource.mu.Lock()
				// If the alarm has been cleared, remove it from the cache
				if alarm.GetClearInfo() != nil {
					delete(dataSource.AlarmCache, alarmID)
					dataSource.mu.Unlock()
					// Skip adding cleared alarms to streaming buffer
					continue
				}

				// Update the cache: merge incoming alarm data onto the existing cached entry
				// so that fields only sent in TRIGGERED/SEVERITY_INCREASED (e.g. mostSevereValue)
				// are not lost when VALUE_UPDATED notifications arrive with partial data.
				if existing, ok := dataSource.AlarmCache[alarmID]; ok {
					merged := proto.Clone(existing).(*alarms.AlarmData)
					proto.Merge(merged, alarm)
					// When an alarm is unshelved, Yamcs sends a notification with no shelveInfo.
					// proto.Merge does not clear existing fields, so we must explicitly clear
					// ShelveInfo when the notification type is UNSHELVED.
					if alarm.GetNotificationType() == alarms.AlarmNotificationType_UNSHELVED {
						merged.ShelveInfo = nil
					}
					dataSource.AlarmCache[alarmID] = merged
				} else {
					dataSource.AlarmCache[alarmID] = alarm
				}
				dataSource.mu.Unlock()
			}
		}
	}
}

func (mux *Multiplexer) Dispose() {
	for _, host := range mux.Hosts {
		if host.Client != nil {
			host.Client.CloseWebSocketConnection()
		}
	}
	mux.Hosts = make(map[string]*YamcsHost)
	mux.Endpoints = make(map[string]*YamcsEndpoint)
}

// GetClient gets or creates a YamcsClient for the given host ID.
func (mux *Multiplexer) GetClient(hostID string) (*client.YamcsClient, error) {

	host, exists := mux.Hosts[hostID]
	if !exists {
		if err := mux.SetupHost(hostID); err != nil {
			return nil, err
		}
		host = mux.Hosts[hostID]
	}

	if host.Client == nil {
		return nil, exception.New("Unexpected error retrieving Yamcs client", "CONNECTION_CLIENT_NOT_FOUND")
	}

	if !host.Client.IsWebSocketConnected() {
		if err := host.Client.EstablishWebSocketConnection(); err != nil {
			return nil, err
		}
	}

	return host.Client, nil
}
