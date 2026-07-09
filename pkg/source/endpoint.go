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
