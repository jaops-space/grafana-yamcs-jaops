package source

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// connectManagerInitialBackoff/connectManagerMaxBackoff bound how aggressively
// a host's background connection manager (see runConnectionManager) retries a
// failing dial: it starts at the initial backoff and doubles up to the max,
// resetting back to the initial value after any successful connect.
const (
	connectManagerInitialBackoff = 1 * time.Second
	connectManagerMaxBackoff     = 30 * time.Second
)

type YamcsHostInstance struct {
	Instance   client.Instance
	Processors map[string]client.Processor
}

// YamcsHost represents a Yamcs server connection along with its instances and processors.
//
// Locking summary (three distinct locks are involved in this subsystem;
// none of them substitute for another):
//   - Multiplexer.SyncMux guards membership of the Multiplexer's Hosts/
//     Endpoints maps (which hosts/endpoints exist), not anything inside a
//     given *YamcsHost.
//   - YamcsHost.connectMu serializes connect+resolve+subscribe work for THIS
//     host only (dial, list instances, set up endpoint subscriptions), so a
//     slow/unreachable host never blocks the same work for unrelated,
//     healthy hosts. Held for the entire connect transition, including the
//     onConnected callback - see runConnectionManager/connectHostSync.
//   - YamcsHost.mu guards the contents of Instances/its Processors maps
//     (read by connectEndpoint, written by GetProcessorListener's callback
//     and by resolveHostInstances).
type YamcsHost struct {
	mu            sync.RWMutex
	Client        *client.YamcsClient
	Instances     map[string]*YamcsHostInstance
	Configuration *config.YamcsHostConfiguration

	// connectMu serializes connect+resolve work for THIS host only (used by
	// both the background connection manager and Multiplexer.ConnectSync), so
	// a slow/unreachable host never blocks connect activity for unrelated,
	// healthy hosts.
	connectMu sync.Mutex

	// connectRequest is a non-blocking "please try to (re)connect" mailbox for
	// this host's background connection manager. Callers that need this host
	// connected (SubscribeStream, RunStream) never dial it themselves - they
	// call RequestConnect(), which just nudges the manager, and immediately
	// check/return based on current state. The manager owns all dialing,
	// pacing and backoff for this host.
	connectRequest chan struct{}

	// stopManager, once closed, tells this host's connection manager goroutine
	// to exit. Closed by the Multiplexer that started it (see Dispose).
	stopManager chan struct{}

	managerOnce sync.Once
}

func (host *YamcsHost) Name() string {
	return host.Configuration.DisplayName()
}

// retreive Host client
func (host *YamcsHost) GetClient() *client.YamcsClient {

	return host.Client

}

// dial performs the actual blocking network connect for this host. It is only
// ever called from within a connectMu-guarded section: either the background
// connection manager (see runConnectionManager) for the live datasource path,
// or Multiplexer.ConnectSync for the one-shot health-check path. No other code
// should call this directly - request a connect via RequestConnect() instead.
func (host *YamcsHost) dial(ctx context.Context) error {

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

// RequestConnect asks this host's background connection manager to (re)try
// connecting soon. It never blocks and never dials the network itself - it
// just nudges the manager goroutine, which owns all pacing/backoff for this
// host. Safe to call as often as needed (e.g. once per failed
// SubscribeStream/RunStream call): concurrent requests coalesce into a single
// pending wake-up.
//
// If the manager hasn't been started for this host (e.g. a throwaway
// health-check Multiplexer that only uses ConnectSync), this is a harmless
// no-op.
func (host *YamcsHost) RequestConnect() {
	if host.connectRequest == nil {
		return
	}
	select {
	case host.connectRequest <- struct{}{}:
	default:
		// A request is already pending; the manager will pick it up.
	}
}

// startConnectionManager launches (once) the background goroutine that owns
// all connect attempts, backoff and reconnection for this host. onConnected is
// invoked (with connectMu held) after a successful dial, to list instances and
// set up endpoint subscriptions - see Multiplexer.finishHostConnect.
func (host *YamcsHost) startConnectionManager(ctx context.Context, hostID string, onConnected func(ctx context.Context, hostID string, host *YamcsHost)) {
	host.managerOnce.Do(func() {
		host.connectRequest = make(chan struct{}, 1)
		host.stopManager = make(chan struct{})
		go host.runConnectionManager(ctx, hostID, onConnected)
		// Kick off an initial connect attempt immediately rather than waiting
		// idle for the first RequestConnect() call.
		host.RequestConnect()
	})
}

// stopConnectionManager signals this host's manager goroutine (if any) to
// exit. Safe to call even if the manager was never started.
func (host *YamcsHost) stopConnectionManager() {
	if host.stopManager != nil {
		close(host.stopManager)
	}
}

// runConnectionManager is the single owner of connect/reconnect activity for
// this host. It idles while connected (waking on an explicit reconnect
// request or the client signalling disconnection), and actively retries with
// exponential backoff while disconnected. Callers never dial directly; they
// only ever call RequestConnect().
func (host *YamcsHost) runConnectionManager(ctx context.Context, hostID string, onConnected func(ctx context.Context, hostID string, host *YamcsHost)) {
	backoff := connectManagerInitialBackoff

	for {
		if host.IsConnected() {
			// Idle until something worth reacting to happens: an explicit
			// request (e.g. a stream about to give up), or the underlying
			// client telling us it just disconnected.
			var disconnected <-chan struct{}
			if cli := host.GetClient(); cli != nil {
				disconnected = cli.Disconnected()
			}
			select {
			case <-host.stopManager:
				return
			case <-host.connectRequest:
			case <-disconnected:
			}
			continue
		}

		host.connectMu.Lock()
		var err error
		if !host.IsConnected() { // re-check under lock; may have raced with ConnectSync
			err = host.dial(ctx)
			if err == nil && onConnected != nil {
				// Runs with connectMu still held, so this connect transition's
				// instance-list/subscription setup can never interleave with
				// another connect attempt for this same host - matching the
				// guarantee ConnectSync/connectHostSync gives its callers.
				onConnected(ctx, hostID, host)
			}
		}
		host.connectMu.Unlock()

		if err == nil {
			backoff = connectManagerInitialBackoff
			continue
		}

		backend.Logger.Warn("Host connect attempt failed, will retry", "host", host.Name(), "error", err, "retryIn", backoff)

		select {
		case <-host.stopManager:
			return
		case <-time.After(backoff):
		case <-host.connectRequest:
			// Someone explicitly asked for a sooner retry; drain and loop
			// immediately instead of waiting out the rest of the backoff.
		}

		if backoff < connectManagerMaxBackoff {
			backoff *= 2
			if backoff > connectManagerMaxBackoff {
				backoff = connectManagerMaxBackoff
			}
		}
	}
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

		hostInstance, ok := host.Instances[instanceName]
		if !ok {
			// Instance was removed (e.g. a reconnect re-listed instances and
			// this one no longer exists) since this listener was registered.
			// Nothing to update.
			backend.Logger.Warn("Processor update for unknown instance, dropping", "host", host.Name(), "instance", instanceName, "processor", processorName)
			return
		}

		// Store the new snapshot (update), not the original processor this
		// listener closed over - otherwise every update would just
		// re-write the stale initial state forever.
		hostInstance.Processors[processorName] = update

	}
}
