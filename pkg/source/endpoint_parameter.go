package source

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

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
	mu sync.Mutex

	parameter *ParameterDemand

	Path   string
	Buffer []client.ParameterValue
}

// GetChannelParameterListener returns a function to listen for parameter updates.
func (ep *YamcsEndpoint) getChannelParameterListener() client.ParameterListener {
	return func(parameter string, value *pvalue.ParameterValue) error {

		ep.mu.Lock()
		defer ep.mu.Unlock()

		paramDemand, err := ep.getOrCreateParameterDemand(context.Background(), parameter)
		if err != nil {
			return err
		}

		streamDemands := paramDemand.Streams
		paramDemand.LastReceived = time.Now()

		if value.GetAcquisitionStatus() != pvalue.AcquisitionStatus_ACQUIRED {
			backend.Logger.Debug("Ignoring parameter value", "parameter", parameter, "status", value.GetAcquisitionStatus())
			return nil
		}

		for _, streamDemand := range streamDemands {
			streamDemand.mu.Lock()
			streamDemand.Buffer = append(streamDemand.Buffer, value)
			streamDemand.mu.Unlock()
		}
		return nil

	}
}

// RequestNewParameterStream adds a new parameter stream to the endpoint.
func (ep *YamcsEndpoint) RequestNewParameterStream(ctx context.Context, name string, path string) error {

	ep.mu.Lock()
	defer ep.mu.Unlock()

	_, err := ep.getOrCreateParameterDemand(ctx, name)
	if err != nil {
		return err
	}

	ep.Parameters[name].Streams[path] = &ParameterStreamDemand{
		parameter: ep.Parameters[name],
		Path:      path,
		Buffer:    make([]*pvalue.ParameterValue, 0),
	}

	subscription, err := ep.getParameterSubscription(ctx)
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
func (ep *YamcsEndpoint) GetAndClearParameterStreamBuffer(parameter string, path string) []client.ParameterValue {

	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.Parameters[parameter] == nil || ep.Parameters[parameter].Streams[path] == nil {
		return nil
	}
	stream := ep.Parameters[parameter].Streams[path]

	stream.mu.Lock()
	defer stream.mu.Unlock()

	buf := stream.Buffer
	out := make([]client.ParameterValue, len(buf))
	copy(out, buf)
	stream.Buffer = stream.Buffer[:0]
	return out

}

// WithdrawParameterStreamRequest removes a parameter stream request.
func (ep *YamcsEndpoint) WithdrawParameterStreamRequest(ctx context.Context, name string, path string) error {

	ep.mu.Lock()
	defer ep.mu.Unlock()

	_, found := ep.Parameters[name]
	if !found {
		return nil
	}
	client, err := ep.GetClient()
	if err != nil {
		return err
	}

	delete(ep.Parameters[name].Streams, path)
	if len(ep.Parameters[name].Streams) == 0 && client != nil && client.IsWebSocketConnected() {
		subscription, err := ep.getParameterSubscription(ctx)
		if err != nil {
			return err
		}
		subscription.Remove(name)
	}
	return nil
}

// GetParameterDemand retrieves or initializes a ParameterDemand.
func (ep *YamcsEndpoint) getOrCreateParameterDemand(ctx context.Context, parameter string) (*ParameterDemand, error) {

	if ep.Parameters[parameter] == nil {

		client, err := ep.GetClient()
		if err != nil {
			return nil, err
		}
		unit := ""
		thresholds := make([]*data.Threshold, 0)

		paramInfo, err := client.GetParameter(ctx, ep.GetInstanceName(), parameter)
		if err != nil {
			return nil, err
		}
		paramType := paramInfo.GetType()
		unitSet := paramType.GetUnitSet()
		thresholds = tools.ConvertAlarmInfoToThresholds(paramType.GetDefaultAlarm())
		if len(unitSet) > 0 {
			unit = unitSet[0].GetUnit()
			backend.Logger.Debug("found unit", "parameter", parameter, "unit", unit)
		}

		ep.Parameters[parameter] = &ParameterDemand{
			endpoint:   ep,
			Name:       parameter,
			Unit:       unit,
			Thresholds: thresholds,
			Streams:    make(map[string]*ParameterStreamDemand),
		}
	}
	return ep.Parameters[parameter], nil
}

// GetParameterSubscription retrieves or creates a parameter subscription.
func (ep *YamcsEndpoint) getParameterSubscription(ctx context.Context) (*client.ParameterSubscription, error) {

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	for _, subscription := range client.ParameterSubscriptions {
		if subscription.Instance == ep.GetInstanceName() && subscription.Processor == ep.GetProcessorName() {
			return subscription, nil
		}
	}
	instance, err := ep.GetInstance()
	if err != nil {
		return nil, err
	}
	processor, err := ep.GetProcessor()
	if err != nil {
		return nil, err
	}
	subscription, err := client.CreateParameterSubscription(ctx, instance, processor)
	if err != nil {
		return nil, err
	}
	subscription.SetListener(ep.getChannelParameterListener())
	return subscription, nil
}

func (endpoint *YamcsEndpoint) SetUnitAndThresholds(ctx context.Context, parameter string, frame *data.Frame) {

	endpoint.mu.Lock()
	defer endpoint.mu.Unlock()

	parameterDemand, err := endpoint.getOrCreateParameterDemand(ctx, parameter)
	if err != nil {
		backend.Logger.Error("could not set units and thresholds", "error", err)
		return
	}

	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeGraph}

	for _, field := range frame.Fields {
		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.Unit = parameterDemand.Unit
		field.Config.Thresholds = &data.ThresholdsConfig{
			Mode:  data.ThresholdsModeAbsolute,
			Steps: make([]data.Threshold, 0, len(parameterDemand.Thresholds)),
		}
		for _, t := range parameterDemand.Thresholds {
			field.Config.Thresholds.Steps = append(field.Config.Thresholds.Steps, *t)
		}
	}
}
