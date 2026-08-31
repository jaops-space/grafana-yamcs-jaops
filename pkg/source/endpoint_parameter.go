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
		started := time.Now()
		streamCount := 0
		defer func() {
			if ep.ParameterProcessObserver != nil && streamCount > 0 {
				ep.ParameterProcessObserver(parameter, streamCount, time.Since(started))
			}
		}()

		paramDemand, err := ep.getOrCreateParameterDemand(context.Background(), parameter)
		if err != nil {
			return err
		}

		ep.mu.Lock()
		streamDemands := make([]*ParameterStreamDemand, 0, len(paramDemand.Streams))
		for _, streamDemand := range paramDemand.Streams {
			streamDemands = append(streamDemands, streamDemand)
		}
		streamCount = len(streamDemands)
		paramDemand.LastReceived = time.Now()
		ep.mu.Unlock()

		status := value.GetAcquisitionStatus()
		if status != pvalue.AcquisitionStatus_ACQUIRED && status != pvalue.AcquisitionStatus_EXPIRED {
			backend.Logger.Debug("Ignoring parameter value", "parameter", parameter, "status", status)
			return nil
		}
		if status == pvalue.AcquisitionStatus_EXPIRED {
			backend.Logger.Debug(
				"Received expired parameter value",
				"parameter", parameter,
				"streamCount", streamCount,
				"generationTime", value.GetGenerationTime().AsTime(),
				"acquisitionTime", value.GetAcquisitionTime().AsTime(),
				"expireMillis", value.GetExpireMillis(),
			)
		}

		receivedAt := time.Now()
		for _, streamDemand := range streamDemands {
			streamDemand.mu.Lock()
			streamDemand.Buffer = append(streamDemand.Buffer, value)
			streamDemand.mu.Unlock()
			if ep.ParameterBufferObserver != nil {
				ep.ParameterBufferObserver(parameter, streamDemand.Path, receivedAt)
			}
		}
		return nil

	}
}

// RequestNewParameterStream adds a new parameter stream to the endpoint.
func (ep *YamcsEndpoint) RequestNewParameterStream(ctx context.Context, name string, path string) error {

	demand, err := ep.getOrCreateParameterDemand(ctx, name)
	if err != nil {
		return err
	}

	ep.mu.Lock()
	demand.Streams[path] = &ParameterStreamDemand{
		parameter: demand,
		Path:      path,
		Buffer:    make([]*pvalue.ParameterValue, 0),
	}
	ep.mu.Unlock()

	subscription, err := ep.getParameterSubscription(ctx)
	if err != nil {
		backend.Logger.Error("Error getting parameter subscription", "error", err)
		return err
	}

	if !subscription.Has(name) {
		backend.Logger.Debug("Adding parameter to subscription", "parameter", name)
		subscription.Add(name)
	}

	for name := range subscription.ActiveSubscriptions {
		backend.Logger.Debug("Current subscription", "parameter", name)
	}

	return nil
}

// GetParameterStreamBuffer retrieves the buffer for a specific parameter stream.
func (ep *YamcsEndpoint) GetAndClearParameterStreamBuffer(parameter string, path string) []client.ParameterValue {

	ep.mu.RLock()
	paramDemand := ep.Parameters[parameter]
	if paramDemand == nil {
		ep.mu.RUnlock()
		return nil
	}
	stream := paramDemand.Streams[path]
	ep.mu.RUnlock()

	if stream == nil {
		return nil
	}

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
	demand, found := ep.Parameters[name]
	if !found {
		ep.mu.Unlock()
		return nil
	}
	delete(demand.Streams, path)
	streamsEmpty := len(demand.Streams) == 0
	ep.mu.Unlock()

	client, err := ep.GetClient()
	if err != nil {
		return err
	}

	if streamsEmpty && client != nil && client.IsWebSocketConnected() {
		subscription, err := ep.getParameterSubscription(ctx)
		if err != nil {
			return err
		}
		subscription.Remove(name)
	}
	return nil
}

// GetParameterDemand retrieves or initializes a ParameterDemand. It manages
// its own locking (parameterDemandMu for the create race, mu for the map
// itself) so callers must not hold mu around it - the HTTP lookup below can
// take a while, and holding mu across it would stall unrelated endpoint
// state (alarms, events, ...) for the duration.
func (ep *YamcsEndpoint) getOrCreateParameterDemand(ctx context.Context, parameter string) (*ParameterDemand, error) {

	ep.mu.RLock()
	demand := ep.Parameters[parameter]
	ep.mu.RUnlock()
	if demand != nil {
		return demand, nil
	}

	// Only one goroutine may fetch/create the demand for this endpoint at a
	// time. Scoped separately from mu so it never blocks unrelated endpoint
	// state.
	ep.parameterDemandMu.Lock()
	defer ep.parameterDemandMu.Unlock()

	// Re-check now that we hold the create-lock: another goroutine may have
	// created it while we were waiting.
	ep.mu.RLock()
	demand = ep.Parameters[parameter]
	ep.mu.RUnlock()
	if demand != nil {
		return demand, nil
	}

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	unit := ""

	paramInfo, err := client.GetParameter(ctx, ep.GetInstanceName(), parameter)
	if err != nil {
		return nil, err
	}
	paramType := paramInfo.GetType()
	unitSet := paramType.GetUnitSet()
	thresholds := tools.ConvertAlarmInfoToThresholds(paramType.GetDefaultAlarm())
	if len(unitSet) > 0 {
		unit = unitSet[0].GetUnit()
		backend.Logger.Debug("found unit", "parameter", parameter, "unit", unit)
	}

	demand = &ParameterDemand{
		endpoint:   ep,
		Name:       parameter,
		Unit:       unit,
		Thresholds: thresholds,
		Streams:    make(map[string]*ParameterStreamDemand),
	}

	ep.mu.Lock()
	ep.Parameters[parameter] = demand
	ep.mu.Unlock()

	return demand, nil
}

// GetParameterSubscription retrieves or creates a parameter subscription. It
// does not touch mu - subscription lookup/creation is entirely guarded by
// the client's own subsMu (for the map) and parameterSubscribeMu (for the
// create race), so a slow/stuck subscribe attempt never blocks unrelated
// endpoint state guarded by mu.
func (ep *YamcsEndpoint) getParameterSubscription(ctx context.Context) (*client.ParameterSubscription, error) {

	client, err := ep.GetClient()
	if err != nil {
		return nil, err
	}
	if subscription, found := client.FindParameterSubscription(ep.GetInstanceName(), ep.GetProcessorName()); found {
		return subscription, nil
	}

	// Only one goroutine may attempt to create the parameter subscription
	// for this endpoint at a time.
	ep.parameterSubscribeMu.Lock()
	defer ep.parameterSubscribeMu.Unlock()

	if subscription, found := client.FindParameterSubscription(ep.GetInstanceName(), ep.GetProcessorName()); found {
		return subscription, nil
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

	parameterDemand, err := endpoint.getOrCreateParameterDemand(ctx, parameter)
	if err != nil {
		backend.Logger.Error("could not set units and thresholds", "error", err)
		return
	}

	field, _ := frame.FieldByName(parameter)
	if field == nil {
		backend.Logger.Debug("could not set units and thresholds; parameter field not found", "parameter", parameter)
		return
	}
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
