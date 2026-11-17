package source

import (
	"context"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/instances"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
)

// discoveryInterval is how often a host with DiscoverInstances lists its
// instances again while connected (new instances, instances moved to it from
// another host of the datasource).
var discoveryInterval = 30 * time.Second

// hasRunningInstance reports whether the host is connected and runs the
// instance, as of its last listing.
func (host *YamcsHost) hasRunningInstance(name string) bool {
	if !host.IsConnected() {
		return false
	}
	host.mu.RLock()
	defer host.mu.RUnlock()
	hi, ok := host.Instances[name]
	return ok && hi.Instance.GetState() == instances.YamcsInstance_RUNNING
}

// discoverEndpoints makes an endpoint for each running instance of a host
// with DiscoverInstances that does not have one yet. The endpoint's ID is
// the instance's name or its alias, and it uses the default (realtime)
// processor. If a discovered endpoint's instance is gone from its old host
// (connected, but no longer listing it) and runs on this one, the endpoint
// is moved here. Endpoints configured by hand are never replaced. Called
// after resolveHostInstances; returns the new endpoints, not yet connected.
func (mux *Multiplexer) discoverEndpoints(hostID string, host *YamcsHost) []hostEndpoint {
	if host.Configuration == nil || !host.Configuration.DiscoverInstances {
		return nil
	}
	host.mu.RLock()
	names := make([]string, 0, len(host.Instances))
	for name, hi := range host.Instances {
		if hi.Instance.GetState() == instances.YamcsInstance_RUNNING {
			names = append(names, name)
		}
	}
	host.mu.RUnlock()
	sort.Strings(names)

	mux.SyncMux.Lock()
	defer mux.SyncMux.Unlock()
	var made []hostEndpoint
	for _, name := range names {
		id := host.Configuration.EndpointID(name)
		if old, ok := mux.Endpoints[id]; ok {
			if !old.Discovered || old.GetHost() == host {
				continue
			}
			if oldHost := old.GetHost(); oldHost != nil && oldHost.hasRunningInstance(old.Configuration.Instance) {
				backend.Logger.Warn("Instance discovered on two hosts, keeping the first", "endpoint", id, "host", host.Name(), "other", oldHost.Name())
				continue
			}
			backend.Logger.Info("Discovered endpoint moved", "endpoint", id, "host", host.Name())
		}
		cfg := &config.YamcsEndpointConfiguration{ID: id, Name: id, Host: hostID, Instance: name}
		ep := mux.newEndpoint(id, cfg, host)
		ep.Discovered = true
		mux.Endpoints[id] = ep
		made = append(made, hostEndpoint{ID: id, Endpoint: ep})
	}
	return made
}

// runDiscovery lists a discovering host's instances every discoveryInterval
// while it is connected and sets up the endpoints it finds, until the host's
// connection manager stops.
func (mux *Multiplexer) runDiscovery(hostID string, host *YamcsHost) {
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-host.stopManager:
			return
		case <-ticker.C:
		}
		if !host.IsConnected() {
			continue
		}
		mux.refreshHost(hostID, host)
	}
}

func (mux *Multiplexer) refreshHost(hostID string, host *YamcsHost) {
	host.connectMu.Lock()
	defer host.connectMu.Unlock()
	cli := host.GetClient()
	if cli == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := resolveHostInstances(ctx, host, cli); err != nil {
		backend.Logger.Warn("Could not list instances", "host", host.Name(), "error", err)
		return
	}
	errs := map[string]error{}
	for _, e := range mux.discoverEndpoints(hostID, host) {
		mux.connectEndpoint(ctx, e.ID, e.Endpoint, host, cli, true, errs)
	}
	for id, err := range errs {
		backend.Logger.Warn("Could not set up discovered endpoint", "endpoint", id, "host", host.Name(), "error", err)
	}
}

// Labels returns the labels of the endpoint's Yamcs instance (e.g. Oasis's
// mdb, mdbVersion, config), as of the host's last listing.
func (ep *YamcsEndpoint) Labels() map[string]string {
	host := ep.GetHost()
	if host == nil || ep.Configuration == nil {
		return nil
	}
	host.mu.RLock()
	defer host.mu.RUnlock()
	hi, ok := host.Instances[ep.Configuration.Instance]
	if !ok {
		return nil
	}
	return hi.Instance.GetLabels()
}

// ListEndpoints returns the endpoints by ID: configured and discovered.
func (mux *Multiplexer) ListEndpoints() map[string]*YamcsEndpoint {
	mux.SyncMux.RLock()
	defer mux.SyncMux.RUnlock()
	out := make(map[string]*YamcsEndpoint, len(mux.Endpoints))
	for id, ep := range mux.Endpoints {
		out[id] = ep
	}
	return out
}

// Discovers reports whether a host of the configuration discovers instances.
func Discovers(cfg *config.YamcsPluginConfiguration) bool {
	for _, h := range cfg.Hosts {
		if h != nil && h.DiscoverInstances {
			return true
		}
	}
	return false
}
