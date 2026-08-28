package source

import (
	"errors"
	"sync"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// YamcsEndpoint represents an endpoint for Yamcs communication.
type YamcsEndpoint struct {
	Multiplexer *Multiplexer
	Host        *YamcsHost

	mu sync.RWMutex

	ID                    string
	Parameters            map[string]*ParameterDemand
	Events                map[string]chan *events.Event
	CommandHistorySignals map[string]CommandHistorySignal
	Alarms                map[string][]*alarms.AlarmData
	AlarmSignals          map[string]chan struct{}
	LinkSignals           map[string]LinkSignal
	AlarmCache            map[string]*alarms.AlarmData // Cache of all active alarms by ID
	GlobalAlarmStatus     *alarms.GlobalAlarmStatus

	CurrentTime          time.Time
	CurrentTimeUpdatedAt time.Time

	ParameterProcessObserver func(parameter string, streamCount int, elapsed time.Duration)
	ParameterBufferObserver  func(parameter string, path string, receivedAt time.Time)

	Configuration *config.YamcsEndpointConfiguration
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
