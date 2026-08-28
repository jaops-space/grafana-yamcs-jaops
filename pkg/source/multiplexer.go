package source

import (
	"context"
	"fmt"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
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

// hostEndpoint pairs an endpoint's config ID with the endpoint itself, for
// code that needs to iterate a host's endpoints and still report errors by ID.
type hostEndpoint struct {
	ID       string
	Endpoint *YamcsEndpoint
}

// NewMultiplexer creates a fresh multiplexer with a connection manager.
func NewMultiplexer(cfg *config.YamcsPluginConfiguration, seccfg *config.YamcsSecureConfiguration) (*Multiplexer, error) {
	return NewMultiplexerWithContext(context.Background(), cfg, seccfg)
}

func NewMultiplexerWithContext(ctx context.Context, cfg *config.YamcsPluginConfiguration, seccfg *config.YamcsSecureConfiguration) (*Multiplexer, error) {

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

		yamcsClient, err := client.NewYamcsClientWithContext(ctx, hostCfg.Path, tlsConfig, creds)
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

// StartConnectionManagers launches (once) each host's background connection
// manager, which owns all dialing/backoff/reconnection for that host from
// then on. This is the entry point used by the live datasource path: it
// returns immediately without blocking on any network activity. Individual
// hosts connect (and their endpoints get resolved/subscribed) asynchronously,
// as soon as each host's manager succeeds.
//
// Managers run off context.Background() rather than any caller-supplied
// context: NewDatasource's ctx is scoped to the single request that happened
// to trigger instance creation and is canceled as soon as that request
// completes, which would otherwise cancel every future dial attempt for this
// host's entire lifetime. Managers still stop cleanly via
// stopConnectionManager() (see Dispose).
//
// SubscribeStream/RunStream never call Connect()/dial a host directly - they
// call YamcsEndpoint.EnsureReady(), which checks current state and, if the
// host isn't connected yet, calls host.RequestConnect() (a non-blocking nudge
// to the manager) before returning a fast error. This way, one slow or
// unreachable host can never stall requests for any other host, and repeated
// requests for a broken host never trigger their own redundant network dials
// - the manager is the single owner of that host's retry pacing.
func (mux *Multiplexer) StartConnectionManagers() {
	mux.SyncMux.RLock()
	hosts := make(map[string]*YamcsHost, len(mux.Hosts))
	for hostID, host := range mux.Hosts {
		hosts[hostID] = host
	}
	mux.SyncMux.RUnlock()

	for hostID, host := range hosts {
		host.startConnectionManager(context.Background(), hostID, mux.finishHostConnect)
	}
}

// resolveHostInstances lists a host's instances/processors from Yamcs and
// stores them on the host (replacing any previously known set), so
// EnsureReady/connectEndpoint can resolve them purely from memory afterwards.
// Called once per successful connect, from both the live (finishHostConnect)
// and one-shot (connectHostSync) paths.
func resolveHostInstances(ctx context.Context, host *YamcsHost, cli *client.YamcsClient) error {
	instances, err := cli.ListInstances(ctx)
	if err != nil {
		return err
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	for _, instance := range instances {
		hostInstance := &YamcsHostInstance{
			Instance:   instance,
			Processors: map[string]client.Processor{},
		}
		for _, processor := range instance.Processors {
			hostInstance.Processors[processor.GetName()] = processor
		}
		host.Instances[instance.GetName()] = hostInstance
	}
	return nil
}

// finishHostConnect lists instances/processors and sets up endpoint
// subscriptions for a host that has just (re)connected. It is called by each
// host's connection manager, with that host's connectMu held, immediately
// after a successful dial - the same "resolve once per connect transition"
// behavior the previous synchronous implementation had.
func (mux *Multiplexer) finishHostConnect(ctx context.Context, hostID string, host *YamcsHost) {
	cli := host.GetClient()
	if cli == nil {
		return
	}

	if err := resolveHostInstances(ctx, host, cli); err != nil {
		backend.Logger.Warn("Could not list instances after connect", "host", host.Name(), "error", err)
		return
	}

	mux.SyncMux.RLock()
	var hostEndpoints []hostEndpoint
	for endpointID, endpoint := range mux.Endpoints {
		if endpoint.GetHost() == host {
			hostEndpoints = append(hostEndpoints, hostEndpoint{ID: endpointID, Endpoint: endpoint})
		}
	}
	mux.SyncMux.RUnlock()

	endpointErrors := map[string]error{}
	for _, e := range hostEndpoints {
		mux.connectEndpoint(ctx, e.ID, e.Endpoint, host, cli, true, endpointErrors)
	}
	for endpointID, err := range endpointErrors {
		backend.Logger.Warn("Could not set up endpoint after host connect", "endpoint", endpointID, "host", host.Name(), "error", err)
	}
}

// ConnectSync connects to all disconnected hosts and endpoints and sets up
// initial subscriptions, blocking until every host has either succeeded or
// failed once. Initial subscriptions can be skipped by setting
// subscribe=false. Returns maps of all errors in hosts and endpoints; the op
// is successful when both maps are empty.
//
// This is a one-shot, synchronous variant intended for callers that need an
// immediate, blocking answer - currently only the datasource health check
// ("Save & Test"), which creates a throwaway Multiplexer purely to test
// connectivity and disposes of it right after. It does NOT start a background
// connection manager and must not be used on the live datasource's
// long-lived Multiplexer - use StartConnectionManagers for that instead.
//
// Locking note: each host's connect+subscribe work is guarded by that host's
// own mutex (YamcsHost.connectMu) rather than a single lock shared across the
// whole Multiplexer, so unreachable hosts here still can't block each other.
func (mux *Multiplexer) ConnectSync(ctx context.Context, subscribe bool) (map[string]error, map[string]error) {

	mux.SyncMux.RLock()
	hosts := make(map[string]*YamcsHost, len(mux.Hosts))
	for hostID, host := range mux.Hosts {
		hosts[hostID] = host
	}
	endpointsByHost := make(map[*YamcsHost][]hostEndpoint)
	for endpointID, endpoint := range mux.Endpoints {
		endpointHost := endpoint.GetHost()
		endpointsByHost[endpointHost] = append(endpointsByHost[endpointHost], hostEndpoint{ID: endpointID, Endpoint: endpoint})
	}
	mux.SyncMux.RUnlock()

	hostErrors := map[string]error{}
	endpointErrors := map[string]error{}

	for hostID, host := range hosts {
		mux.connectHostSync(ctx, hostID, host, endpointsByHost[host], subscribe, hostErrors, endpointErrors)
	}

	// Endpoints whose host could not be resolved at all (nil host) aren't
	// reachable via endpointsByHost keyed by *YamcsHost, so report them here.
	if nilHostEndpoints, ok := endpointsByHost[nil]; ok {
		for _, e := range nilHostEndpoints {
			endpointErrors[e.ID] = exception.New(fmt.Sprintf("host for endpoint %s not found", e.Endpoint.Name()), "MUX_CONNECT_ENDPOINT_NO_HOST")
		}
	}

	return hostErrors, endpointErrors
}

// connectHostSync connects a single host (if not already connected) and, only
// on the call that actually transitions it from disconnected to connected,
// sets up subscriptions for all of that host's endpoints. Used solely by
// ConnectSync; the live path uses each host's background connection manager
// instead (see YamcsHost.runConnectionManager / Multiplexer.finishHostConnect).
func (mux *Multiplexer) connectHostSync(
	ctx context.Context,
	hostID string,
	host *YamcsHost,
	hostEndpoints []hostEndpoint,
	subscribe bool,
	hostErrors map[string]error,
	endpointErrors map[string]error,
) {
	host.connectMu.Lock()
	defer host.connectMu.Unlock()

	if host.IsConnected() {
		// Already connected (and therefore already had its endpoints subscribed)
		// by a previous call. Nothing left to do for this host.
		return
	}

	if err := host.dial(ctx); err != nil {
		hostErrors[hostID] = err
		return
	}

	cli := host.GetClient()
	if cli == nil {
		hostErrors[hostID] = exception.New(fmt.Sprintf("client for %s not found", host.Name()), "MUX_CONNECT_WITHOUT_CLIENT")
		return
	}

	if err := resolveHostInstances(ctx, host, cli); err != nil {
		hostErrors[hostID] = exception.Wrap(fmt.Sprintf("could not list instances for host %s", host.Name()), "MUX_CONNECT_LIST_INSTANCES", err)
		return
	}

	for _, e := range hostEndpoints {
		mux.connectEndpoint(ctx, e.ID, e.Endpoint, host, cli, subscribe, endpointErrors)
	}
}

// connectEndpoint sets up (or reports errors for) a single endpoint's processor
// and parameter subscriptions. It is called from connectHostSync (with that
// host's connectMu held) or from finishHostConnect (called by the background
// connection manager right after a successful connect); either way, it takes
// its own host.mu read lock around reading host.Instances, since that map can
// also be written concurrently by GetProcessorListener's background callback.
func (mux *Multiplexer) connectEndpoint(
	ctx context.Context,
	endpointID string,
	endpoint *YamcsEndpoint,
	endpointHost *YamcsHost,
	cli *client.YamcsClient,
	subscribe bool,
	endpointErrors map[string]error,
) {
	instanceName := endpoint.Configuration.Instance

	endpointHost.mu.RLock()
	hInstance, ok := endpointHost.Instances[instanceName]
	endpointHost.mu.RUnlock()
	if !ok {
		endpointErrors[endpointID] = exception.New(fmt.Sprintf("instance %s not found for endpoint %s", instanceName, endpoint.Name()), "MUX_CONNECT_NO_INSTANCE")
		return
	}
	instance := hInstance.Instance

	var processor client.Processor
	processorName := endpoint.Configuration.Processor
	if processorName == "" {
		processor = cli.GetInstanceDefaultProcessor(hInstance.Instance)
		if processor == nil {
			endpointErrors[endpointID] = exception.New(fmt.Sprintf("endpoint %s is set to default processor, yet host %s has no default processor", endpoint.Name(), instanceName), "MUX_CONNECT_NO_DEFAULT_PROCESSOR")
			return
		}
		processorName = processor.GetName()
		endpoint.Configuration.Processor = processorName // save it
	} else {
		endpointHost.mu.RLock()
		processor, ok = hInstance.Processors[processorName]
		endpointHost.mu.RUnlock()
		if !ok {
			endpointErrors[endpointID] = exception.New(fmt.Sprintf("processor %s not found on instance %s for endpoint %s", processorName, instanceName, endpoint.Name()), "MUX_CONNECT_NO_PROCESSOR")
			return
		}
	}

	if !subscribe {
		return
	}

	prosub, err := cli.CreateProcessorSubscription(ctx, instance, processor)
	if err != nil {
		endpointErrors[endpointID] = exception.Wrap(fmt.Sprintf("could not subscribe to updates on processor %s", processorName), "MUX_CONNECT_SUB_FAIL", err)
		return
	}
	prosub.SetListener(endpointHost.GetProcessorListener(instance, processor))

	// Create a parameter subscription, that will be used to add and remove parameters
	parsub, err := cli.CreateParameterSubscription(ctx, instance, processor)
	if err != nil {
		endpointErrors[endpointID] = exception.Wrap(fmt.Sprintf("could not create parameter subscriptions on %s", processorName), "MUX_CONNECT_SUB_FAIL", err)
		return
	}
	parsub.SetListener(endpoint.getChannelParameterListener())
}

func (mux *Multiplexer) Dispose() {
	for _, host := range mux.Hosts {
		host.stopConnectionManager()
		if host.Client != nil {
			host.Client.Close()
		}
	}
	mux.Hosts = make(map[string]*YamcsHost)
	mux.Endpoints = make(map[string]*YamcsEndpoint)
}

// GetSecureData returns the secure (credential) configuration for a given
// host ID, or nil if unset/not found.
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
