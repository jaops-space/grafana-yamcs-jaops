package source

import (
	"context"
	"fmt"
	"sync"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// Multiplexer manages live parameter subscriptions, ensuring that only one subscription is active per parameter.
// It will automatically terminate subscriptions when they are no longer needed.
// It delegates connection management to ConnectionManager.
type Multiplexer struct {
	Hosts     map[string]*YamcsHost
	Endpoints map[string]*YamcsEndpoint
	Config    *config.YamcsPluginConfiguration
	Secure    *config.YamcsSecureConfiguration

	SyncMux sync.RWMutex
}

// NewMultiplexer creates a fresh multiplexer with a connection manager.
func NewMultiplexer(cfg *config.YamcsPluginConfiguration, seccfg *config.YamcsSecureConfiguration) (*Multiplexer, error) {

	mux := &Multiplexer{
		Hosts:     make(map[string]*YamcsHost),
		Endpoints: make(map[string]*YamcsEndpoint),
		Config:    cfg,
		Secure:    seccfg,
		SyncMux:   sync.RWMutex{},
	}

	// Set up hosts
	for hostID, hostCfg := range cfg.Hosts {
		var tlsConfig corehttp.TLS
		var creds corehttp.Credentials

		if hostCfg.Tls {
			tlsConfig = corehttp.GetTLSConfiguration(!hostCfg.TlsInsecure)
		} else {
			tlsConfig = corehttp.GetNoTLSConfiguration()
		}

		if !hostCfg.Auth {
			creds = &corehttp.NoCredentials{}
		} else {
			username := hostCfg.Username
			secure, found := seccfg.Hosts[hostID]
			if !found {
				return nil, exception.New(fmt.Sprintf("Secure configuration for host %s not found", hostID), "SECURE_CONFIGURATION_NOT_FOUND")
			}
			password := secure.Password
			creds = &corehttp.BasicAuthCredentials{
				Username: username,
				Password: password,
			}
		}

		yamcsClient, err := client.NewYamcsClient(hostCfg.Path, tlsConfig, creds)
		if err != nil {
			return nil, err
		}
		mux.Hosts[hostID] = &YamcsHost{
			Instances:     map[string]*YamcsHostInstance{},
			Configuration: hostCfg,
			Client:        yamcsClient,
		}
	}

	for endpointID, endpointCfg := range cfg.Endpoints {

		endpointHostID := endpointCfg.Host
		host, ok := mux.Hosts[endpointHostID]
		if !ok {
			return nil, exception.New(fmt.Sprintf("Host %s (for endpoint %s) not found", endpointHostID, endpointID), "ENDPOINT_HOST_NOT_FOUND")
		}

		mux.Endpoints[endpointID] = &YamcsEndpoint{
			Configuration:         endpointCfg,
			Multiplexer:           mux,
			Host:                  host,
			Parameters:            make(map[string]*ParameterDemand),
			Events:                make(map[string]chan *events.Event),
			CommandHistorySignals: make(map[string]CommandHistorySignal),
			Alarms:                make(map[string][]*alarms.AlarmData),
			AlarmSignals:          make(map[string]chan struct{}),
			LinkSignals:           make(map[string]LinkSignal),
			AlarmCache:            make(map[string]*alarms.AlarmData),
			ID:                    endpointID,
		}
	}

	return mux, nil
}

func (mux *Multiplexer) GetEndpoint(endpointID string) (*YamcsEndpoint, error) {

	mux.SyncMux.RLock()
	defer mux.SyncMux.RUnlock()

	ep, ok := mux.Endpoints[endpointID]

	if !ok {
		return nil, exception.New(fmt.Sprintf("endpoint %s not found", endpointID), "ENDPOINT_NOT_FOUND")
	}

	return ep, nil

}

// Connect attemps to connect to all disconnected hosts and endpoints and setup initial subscriptions
// Initial subscriptions can be skipped by setting subscribe=false, this is mainly used in health checks
// returns map of all errors in hosts and endpoints, op is sucessful when size of both maps is 0
func (mux *Multiplexer) Connect(ctx context.Context, subscribe bool) (map[string]error, map[string]error) {

	mux.SyncMux.Lock()
	defer mux.SyncMux.Unlock()

	hostErrors := map[string]error{}
	endpointErrors := map[string]error{}

	alreadyConnectedHosts := types.NewSet[*YamcsHost]()

	for hostID, host := range mux.Hosts {

		if host.IsConnected() {
			alreadyConnectedHosts.Add(host)
			continue
		}

		err := host.Connect(ctx)
		if err != nil {
			hostErrors[hostID] = err
			continue
		}

		cli := host.GetClient()
		if cli == nil {
			hostErrors[hostID] = exception.New(fmt.Sprintf("client for %s not found", host.Name()), "MUX_CONNECT_WITHOUT_CLIENT")
			continue
		}

		instances, err := cli.ListInstances(ctx)
		if err != nil {
			hostErrors[hostID] = exception.Wrap(fmt.Sprintf("could not list instances for host %s", host.Name()), "MUX_CONNECT_LIST_INSTANCES", err)
			continue
		}
		for _, instance := range instances {
			host.Instances[instance.GetName()] = &YamcsHostInstance{
				Instance:   instance,
				Processors: map[string]client.Processor{},
			}
			for _, processor := range instance.Processors {
				host.Instances[instance.GetName()].Processors[processor.GetName()] = processor
			}

		}

	}

	for endpointID, endpoint := range mux.Endpoints {

		endpointHost := endpoint.GetHost()
		if endpointHost == nil {
			endpointErrors[endpointID] = exception.New(fmt.Sprintf("host for endpoint %s not found", endpoint.Name()), "MUX_CONNECT_ENDPOINT_NO_HOST")
			continue
		}

		// skip if already connected to host beforehand
		if alreadyConnectedHosts.Exists(endpointHost) {
			continue
		}

		cli := endpointHost.GetClient()
		if cli == nil {
			endpointErrors[endpointID] = exception.New(fmt.Sprintf("client for endpoint %s not found", endpoint.Name()), "MUX_CONNECT_ENDPOINT_NO_CLIENT")
			continue
		}

		if _, hasError := hostErrors[endpointHost.Configuration.ID]; hasError {
			continue
		}

		instanceName := endpoint.Configuration.Instance
		hInstance, ok := endpointHost.Instances[instanceName]
		if !ok {
			endpointErrors[endpointID] = exception.New(fmt.Sprintf("instance %s not found for endpoint %s", instanceName, endpoint.Name()), "MUX_CONNECT_NO_INSTANCE")
			continue
		}
		instance := hInstance.Instance

		var processor client.Processor
		processorName := endpoint.Configuration.Processor
		if processorName == "" {
			processor = cli.GetInstanceDefaultProcessor(hInstance.Instance)
			if processor == nil {
				endpointErrors[endpointID] = exception.New(fmt.Sprintf("endpoint %s is set to default processor, yet host %s has no default processor", endpoint.Name(), instanceName), "MUX_CONNECT_NO_DEFAULT_PROCESSOR")
				continue
			}
			processorName = processor.GetName()
			endpoint.Configuration.Processor = processorName // save it
		} else {
			processor, ok = hInstance.Processors[processorName]
			if !ok {
				endpointErrors[endpointID] = exception.New(fmt.Sprintf("processor %s not found on instance %s for endpoint %s", processorName, instanceName, endpoint.Name()), "MUX_CONNECT_NO_PROCESSOR")
				continue
			}
		}

		if !subscribe {
			continue
		}

		prosub, err := cli.CreateProcessorSubscription(ctx, instance, processor)
		if err != nil {
			endpointErrors[endpointID] = exception.Wrap(fmt.Sprintf("could not subscribe to updates on processor %s", processorName), "MUX_CONNECT_SUB_FAIL", err)
			continue
		}
		prosub.SetListener(endpointHost.GetProcessorListener(instance, processor))

		// Create a parameter subscription, that will be used to add and remove parameters
		parsub, err := cli.CreateParameterSubscription(ctx, instance, processor)
		if err != nil {
			endpointErrors[endpointID] = exception.Wrap(fmt.Sprintf("could not create parameter subscriptions on %s", processorName), "MUX_CONNECT_SUB_FAIL", err)
			continue
		}
		parsub.SetListener(endpoint.getChannelParameterListener())

	}

	return hostErrors, endpointErrors

}

func (mux *Multiplexer) Dispose() {
	for _, host := range mux.Hosts {
		if host.Client != nil {
			host.Client.Close()
		}
	}
	mux.Hosts = make(map[string]*YamcsHost)
	mux.Endpoints = make(map[string]*YamcsEndpoint)
}
