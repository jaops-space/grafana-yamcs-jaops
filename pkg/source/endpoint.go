package source

import (
	"errors"
	"sync"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// YamcsEndpoint represents an endpoint for Yamcs communication.
type YamcsEndpoint struct {
	Multiplexer *Multiplexer
	Host        *YamcsHost

	mu sync.RWMutex

	// subscribeMu guards "find-or-create subscription" races per
	// subscription type below. Each is intentionally separate from mu (and
	// from each other): the actual subscribe attempt can block for the
	// underlying WebSocket's full reply timeout (up to 10s), and previously
	// that wait was made *while holding mu itself* - meaning one slow/stuck
	// subscribe attempt (e.g. alarms) could stall every other RunXStream
	// operation on this endpoint (parameters, events, ...), not just
	// retries of its own kind. These locks keep the "don't create a
	// duplicate subscription" guarantee scoped to just their own
	// subscription type, so mu itself is only ever held for the endpoint's
	// lightweight bookkeeping maps below, never across network I/O.
	parameterDemandMu    sync.Mutex
	parameterSubscribeMu sync.Mutex
	alarmSubscribeMu     sync.Mutex
	eventSubscribeMu     sync.Mutex
	commandHistorySubMu  sync.Mutex
	linkSubscribeMu      sync.Mutex
	timeSubscribeMu      sync.Mutex

	ID         string
	Parameters map[string]*ParameterDemand

	// Events/CommandHistorySignals/LinkSignals are instance-wide broadcast
	// streams: every stream (panel) watching a given type sees the exact
	// same sequence of values. Each is backed by one shared ring buffer
	// (EventsRing/CommandHistoryRing/LinksRing below) that the WebSocket
	// read-loop listener pushes into exactly once per incoming value,
	// regardless of how many stream paths are watching - each demand then
	// tracks only its own read cursor into that shared ring, and a small
	// coalesced "new data" notify channel used to wake its RunXStream
	// goroutine (which otherwise blocks on a channel receive rather than
	// polling a ticker, unlike parameter streams).
	Events                map[string]*BroadcastStreamDemand[*events.Event]
	CommandHistorySignals map[string]*BroadcastStreamDemand[*commanding.CommandHistoryEntry]
	LinkSignals           map[string]*BroadcastStreamDemand[*links.LinkEvent]

	EventsRing         *types.Ring[*events.Event]
	CommandHistoryRing *types.Ring[*commanding.CommandHistoryEntry]
	LinksRing          *types.Ring[*links.LinkEvent]

	Alarms            map[string][]*alarms.AlarmData
	AlarmSignals      map[string]chan struct{}
	AlarmCache        map[string]*alarms.AlarmData // Cache of all active alarms by ID
	GlobalAlarmStatus *alarms.GlobalAlarmStatus

	CurrentTime          time.Time
	CurrentTimeUpdatedAt time.Time

	ParameterProcessObserver func(parameter string, streamCount int, elapsed time.Duration)
	ParameterBufferObserver  func(parameter string, path string, receivedAt time.Time)

	Configuration *config.YamcsEndpointConfiguration
}

// BroadcastStreamDemand is one stream path's view into an instance-wide
// broadcast ring (events, command history, links - see the doc comment on
// YamcsEndpoint's Events/CommandHistorySignals/LinkSignals fields above).
// cursor is only ever read/written by the single RunXStream goroutine that
// owns this path, so it needs no locking of its own, mirroring
// ParameterStreamDemand.
type BroadcastStreamDemand[T any] struct {
	cursor uint64
	notify chan struct{}
}

// newBroadcastStreamDemand creates a demand starting at ring's current
// write position (so it only observes values pushed after it started
// watching, not the ring's entire pre-existing backlog), with a
// coalesced, non-blocking notify channel of capacity
// StreamSignalBufferSize.
func newBroadcastStreamDemand[T any](ring *types.Ring[T]) *BroadcastStreamDemand[T] {
	return &BroadcastStreamDemand[T]{
		cursor: ring.Cursor(),
		notify: make(chan struct{}, StreamSignalBufferSize),
	}
}

func (endpoint *YamcsEndpoint) Name() string {
	return endpoint.Configuration.DisplayName()
}

// GetHost grabs endpoint's host
func (ep *YamcsEndpoint) GetHost() *YamcsHost {
	return ep.Host
}

// GetClient attemps to grab host and its client, returns an error if either failed
func (ep *YamcsEndpoint) GetClient() (*client.YamcsClient, error) {

	host := ep.GetHost()
	if host == nil {
		return nil, errors.New("host not found")
	}

	cli := host.GetClient()
	if cli == nil {
		return nil, errors.New("client not found")
	}
	return cli, nil

}

func (ep *YamcsEndpoint) GetInstance() (client.Instance, error) {

	host := ep.GetHost()
	if host == nil {
		return nil, errors.New("host not found")
	}
	host.mu.RLock()
	defer host.mu.RUnlock()
	hInstance := host.Instances[ep.Configuration.Instance]
	if hInstance == nil || hInstance.Instance == nil {
		return nil, errors.New("instance not found")
	}

	return hInstance.Instance, nil
}

func (ep *YamcsEndpoint) GetProcessor() (client.Processor, error) {

	host := ep.GetHost()
	if host == nil {
		return nil, errors.New("host not found")
	}
	host.mu.RLock()
	defer host.mu.RUnlock()
	hInstance := host.Instances[ep.Configuration.Instance]
	if hInstance == nil {
		return nil, errors.New("instance not found")
	}
	processor := hInstance.Processors[ep.Configuration.Processor]
	if processor == nil {
		return nil, errors.New("processor not found")
	}
	return processor, nil
}

func (ep *YamcsEndpoint) GetInstanceName() string {
	return ep.Configuration.Instance
}
func (ep *YamcsEndpoint) GetProcessorName() string {
	return ep.Configuration.Processor
}

// EnsureReady checks, purely from in-memory state, whether this endpoint is
// ready to serve a request right now: its host must be connected, and its
// configured instance/processor must have actually been resolved on that host
// (populated by the host's background connection manager after a successful
// connect - see YamcsHost.runConnectionManager and Multiplexer.finishHostConnect).
//
// It never performs network I/O itself. If the host isn't connected yet, it
// asks the host's connection manager to try (via RequestConnect, which is a
// non-blocking nudge, not a direct dial) and returns immediately with an
// error - callers should surface that error and let Grafana's own stream
// retry mechanism call back in later, by which point the background manager
// may have succeeded. This keeps SubscribeStream/RunStream from ever blocking
// on - or repeatedly triggering - a network dial themselves, and from ever
// issuing a doomed historical-data request against a host/instance that is
// already known, in memory, to be unavailable.
func (ep *YamcsEndpoint) EnsureReady() error {
	host := ep.GetHost()
	if host == nil {
		return errors.New("host not found")
	}

	if !host.IsConnected() {
		host.RequestConnect()
		return errors.New("host not connected yet; reconnect requested")
	}

	instance, err := ep.GetInstance()
	if err != nil {
		return err
	}

	if ep.Configuration.Processor != "" {
		if _, err := ep.GetProcessor(); err != nil {
			return err
		}
		return nil
	}

	// No explicit processor configured: mirror connectEndpoint's default
	// resolution (first persistent, non-replay processor), using data already
	// fetched by the connect manager - no network call needed here.
	for _, processor := range instance.GetProcessors() {
		if processor.GetPersistent() && !processor.GetReplay() {
			return nil
		}
	}
	return errors.New("no default processor available for instance " + ep.Configuration.Instance)
}
