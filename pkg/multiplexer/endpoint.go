package multiplexer

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

// YamcsEndpoint represents an endpoint for Yamcs communication.
type YamcsEndpoint struct {
	Multiplexer *Multiplexer

	ID         string
	Instance   client.Instance
	Processor  client.Processor
	Parameters map[string]*ParameterDemand
	Events     map[string][]*events.Event
}

// ParameterDemand represents a demand for a specific parameter.
type ParameterDemand struct {
	endpoint *YamcsEndpoint

	LastReceived time.Time
	Name         string
	Unit         string
	Thresholds   []*data.Threshold
	Streams      map[string]*ParameterStreamDemand
}

// ParameterStreamDemand represents a demand for a specific parameter stream.
type ParameterStreamDemand struct {
	parameter *ParameterDemand

	Path   string
	Buffer []client.ParameterValue
}

// GetHostConfiguration retrieves the host configuration for the endpoint.
func (ep *YamcsEndpoint) GetHostConfiguration(name string) *config.YamcsHostConfiguration {
	return ep.Multiplexer.Configuration.Hosts[ep.Multiplexer.Configuration.Endpoints[ep.ID].Host]
}

// GetConfiguration retrieves the source configuration for the endpoint.
func (ep *YamcsEndpoint) GetConfiguration() *config.YamcsEndpointConfiguration {
	return ep.Multiplexer.Configuration.Endpoints[ep.ID]
}

// GetParameterDemand retrieves or initializes a ParameterDemand.
func (ep *YamcsEndpoint) GetParameterDemand(parameter string) *ParameterDemand {

	if ep.Parameters[parameter] == nil {
		client := ep.GetClient()
		unit := ""
		thresholds := make([]*data.Threshold, 0)

		paramInfo, err := client.GetParameter(ep.Instance, parameter)
		if err == nil {
			paramType := paramInfo.GetType()
			unitSet := paramType.GetUnitSet()
			thresholds = tools.ConvertAlarmInfoToThresholds(paramType.GetDefaultAlarm())
			if len(unitSet) > 0 {
				unit = unitSet[0].GetUnit()
			}
		}

		ep.Parameters[parameter] = &ParameterDemand{
			endpoint:   ep,
			Name:       parameter,
			Unit:       unit,
			Thresholds: thresholds,
			Streams:    make(map[string]*ParameterStreamDemand),
		}
	}
	return ep.Parameters[parameter]
}

// GetChannelParameterListener returns a function to listen for parameter updates.
func (ep *YamcsEndpoint) GetChannelParameterListener() func(parameter string, value *pvalue.ParameterValue) {
	return func(parameter string, value *pvalue.ParameterValue) {

		paramDemand := ep.GetParameterDemand(parameter)
		streamDemands := paramDemand.Streams
		paramDemand.LastReceived = time.Now()

		if value.GetAcquisitionStatus() != pvalue.AcquisitionStatus_ACQUIRED {
			backend.Logger.Debug("Ignoring parameter value", "parameter", parameter, "status", value.GetAcquisitionStatus())
			return
		}

		for _, streamDemand := range streamDemands {
			streamDemand.Buffer = append(streamDemand.Buffer, value)
		}

	}
}

// RequestNewParameterStream adds a new parameter stream to the endpoint.
func (ep *YamcsEndpoint) RequestNewParameterStream(name string, path string) error {

	ep.GetParameterDemand(name)

	ep.Parameters[name].Streams[path] = &ParameterStreamDemand{
		parameter: ep.Parameters[name],
		Path:      path,
		Buffer:    make([]*pvalue.ParameterValue, 0),
	}

	subscription, err := ep.GetParameterSubscription()
	if err != nil {
		backend.Logger.Error("Error getting parameter subscription", "error", err)
		return err
	}

	if !subscription.Has(name) {
		backend.Logger.Debug("Adding parameter to subscription", "parameter", name)
		subscription.Add(name)
	}
	backend.Logger.Debug("Current subscriptions", "subscriptions")
	for name := range subscription.ActiveSubscriptions {
		backend.Logger.Debug(name)
	}

	return nil
}

// GetParameterStreamBuffer retrieves the buffer for a specific parameter stream.
func (ep *YamcsEndpoint) GetParameterStreamBuffer(parameter string, path string) []client.ParameterValue {
	if ep.Parameters[parameter].Streams[path] == nil {
		ep.RequestNewParameterStream(parameter, path)
		return ep.Parameters[parameter].Streams[path].Buffer
	}
	return ep.Parameters[parameter].Streams[path].Buffer
}

// ClearParameterStream clears the buffer for a specific parameter stream.
func (ep *YamcsEndpoint) ClearParameterStream(parameter string, path string) {
	if ep.Parameters[parameter].Streams[path] == nil {
		ep.RequestNewParameterStream(parameter, path)
	}
	ep.Parameters[parameter].Streams[path].Buffer = make([]client.ParameterValue, 0)
}

// WithdrawParameterStreamRequest removes a parameter stream request.
func (ep *YamcsEndpoint) WithdrawParameterStreamRequest(name string, path string) error {

	ep.GetParameterDemand(name)
	client := ep.GetClient()

	delete(ep.Parameters[name].Streams, path)
	if len(ep.Parameters[name].Streams) == 0 && client != nil && client.IsWebSocketConnected() {
		subscription, err := ep.GetParameterSubscription()
		if err != nil {
			return err
		}
		subscription.Remove(name)
	}
	return nil
}

// GetClient retrieves the Yamcs client for this endpoint.
func (ep *YamcsEndpoint) GetClient() *client.YamcsClient {
	return ep.Multiplexer.Hosts[ep.GetConfiguration().Host].Client
}

// GetParameterSubscription retrieves or creates a parameter subscription.
func (ep *YamcsEndpoint) GetParameterSubscription() (*client.ParameterSubscription, error) {
	client := ep.GetClient()
	for _, subscription := range client.ParameterSubscriptions {
		if subscription.Instance == ep.Instance.GetName() && subscription.Processor == ep.Processor.GetName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateParameterSubscription(ep.Instance, ep.Processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.GetChannelParameterListener())
	return subscription, nil
}

// RequestEventsStream initiates an event stream subscription.
func (ep *YamcsEndpoint) RequestEventsStream(path string) {
	ep.GetEventsSubscription()
	ep.Events[path] = make([]*events.Event, 0)
}

func (source *YamcsEndpoint) GetEventsSubscription() (*client.EventSubscription, error) {

	client := source.GetClient()
	for _, subscription := range client.EventSubscriptions {
		if subscription.Instance == source.Instance.GetName() {
			return subscription, nil
		}
	}
	subscription, err := client.CreateEventSubscription(source.Instance)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(source.Multiplexer.GetEventListener(source.Instance))
	return subscription, nil

}

func (ep *YamcsEndpoint) GetEventsStream(path string) []*events.Event {

	return ep.Events[path]

}

func (ep *YamcsEndpoint) ClearEventsStream(path string) {

	ep.Events[path] = make([]*events.Event, 0)

}

// WithdrawEventsStreamRequest stops an event stream subscription.
func (ep *YamcsEndpoint) WithdrawEventsStreamRequest(path string) {

	delete(ep.Events, path)
	if len(ep.Events) == 0 {
		client := ep.GetClient()
		for _, subscription := range client.EventSubscriptions {
			if subscription.Instance == ep.Instance.GetName() {
				subscription.Halt()
			}
		}
	}
}

/**


+0x7b\npanic({0xf4d3e0?, 0x1b50d40?})\n\t/home/linuxbrew/.linuxbrew/Cellar/go/1.23.5/libexec/src/runtime/panic.go:785 +0x132\ngithub.com/jaops-space/grafana-yamcs-jaops/pkg/multiplexer.(*YamcsEndpoint).GetConfiguration(...)\n\t/home/kateonbxsh/jaops-grafana/pkg/multiplexer/endpoint.go:48\ngithub.com/jaops-space/grafana-yamcs-jaops/pkg/multiplexer.(*YamcsEndpoint).GetClient(...)\n\t/home/kateonbxsh/jaops-grafana/pkg/multiplexer/endpoint.go:165\ngithub.com/jaops-space/grafana-yamcs-jaops/pkg/plugin.(*Datasource).handleSearchParameters(0xc0004436c0, {0x1272c08, 0xc0005252c0}, 0xc000413400)\n\t/home/kateonbxsh/jaops-grafana/pkg/plugin/resources.go:52

*/
