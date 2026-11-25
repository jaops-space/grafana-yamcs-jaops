package source

import (
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/instances"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

func discoveringHost(id string, aliases map[string]string, names ...string) *YamcsHost {
	h := &YamcsHost{
		Instances:     map[string]*YamcsHostInstance{},
		Configuration: &config.YamcsHostConfiguration{ID: id, Path: id + ":8090", DiscoverInstances: true, Aliases: aliases},
	}
	running := instances.YamcsInstance_RUNNING
	for _, n := range names {
		name := n
		h.Instances[n] = &YamcsHostInstance{
			Instance:   &instances.YamcsInstance{Name: &name, State: &running, Labels: map[string]string{"mdb": "pussim"}},
			Processors: map[string]client.Processor{},
		}
	}
	return h
}

func TestDiscoverEndpoints(t *testing.T) {
	mux := &Multiplexer{Hosts: map[string]*YamcsHost{}, Endpoints: map[string]*YamcsEndpoint{}}
	a := discoveringHost("yamcs-a", map[string]string{"fm1": "sat-9"}, "sat-1", "fm1")
	mux.Hosts["yamcs-a"] = a
	// A configured endpoint with a discovered instance's ID wins.
	mux.Endpoints["sat-1"] = mux.newEndpoint("sat-1", &config.YamcsEndpointConfiguration{Host: "yamcs-a", Instance: "sat-1", Processor: "replay"}, a)

	made := mux.discoverEndpoints("yamcs-a", a)
	if len(made) != 1 || made[0].ID != "sat-9" {
		t.Fatalf("made %v", made)
	}
	ep := mux.Endpoints["sat-9"]
	if !ep.Discovered || ep.Configuration.Instance != "fm1" || ep.Configuration.Host != "yamcs-a" || ep.Labels()["mdb"] != "pussim" {
		t.Errorf("endpoint %+v", ep.Configuration)
	}
	if mux.Endpoints["sat-1"].Discovered || mux.Endpoints["sat-1"].Configuration.Processor != "replay" {
		t.Errorf("configured endpoint replaced")
	}
	if again := mux.discoverEndpoints("yamcs-a", a); len(again) != 0 {
		t.Errorf("discovered again: %v", again)
	}

	// fm1 moves to yamcs-b: a is not connected (its listing no longer counts),
	// so the endpoint follows.
	b := discoveringHost("yamcs-b", map[string]string{"fm1": "sat-9"}, "fm1")
	mux.Hosts["yamcs-b"] = b
	made = mux.discoverEndpoints("yamcs-b", b)
	if len(made) != 1 || mux.Endpoints["sat-9"].GetHost() != b {
		t.Errorf("not moved: %v", made)
	}

	// A host without discovery makes nothing.
	c := discoveringHost("yamcs-c", nil, "sat-3")
	c.Configuration.DiscoverInstances = false
	if made := mux.discoverEndpoints("yamcs-c", c); len(made) != 0 {
		t.Errorf("made %v", made)
	}
}

func TestDiscoverSkipsStopped(t *testing.T) {
	mux := &Multiplexer{Hosts: map[string]*YamcsHost{}, Endpoints: map[string]*YamcsEndpoint{}}
	a := discoveringHost("yamcs-a", nil, "sat-1")
	offline := instances.YamcsInstance_OFFLINE
	a.Instances["sat-1"].Instance.State = &offline
	if made := mux.discoverEndpoints("yamcs-a", a); len(made) != 0 {
		t.Errorf("made %v", made)
	}
}
