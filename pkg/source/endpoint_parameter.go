package source

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
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

	// Ring is the single shared buffer for every stream watching this
	// parameter. The WebSocket read-loop listener pushes each incoming
	// value exactly once here, regardless of how many streams (panels) are
	// watching it - each ParameterStreamDemand then reads whatever is new
	// since its own cursor. This replaces the old design where the
	// listener re-appended the same value into every stream's own
	// mutex-guarded buffer (O(streams) work per value); pushing once here
	// is O(1) per value no matter the fan-out.
	Ring *types.Ring[client.ParameterValue]
}

// ParameterStreamDemand represents a single stream's (i.e. a single panel's)
// view into its parameter's shared Ring. cursor is only ever read/written
// by the one RunParameterStream goroutine that owns this path (via
// GetAndClearParameterStreamBuffer) - never touched by the WebSocket
// read-loop or by any other stream's goroutine - so it needs no locking of
// its own.
type ParameterStreamDemand struct {
	parameter *ParameterDemand

	Path   string
	cursor uint64
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

		// This listener runs synchronously on the WebSocket's single read
		// loop, so it must never block on network I/O - a demand should
		// always already exist by the time a value arrives (created by
		// RequestNewParameterStream before the subscribe request is even
		// sent). If it doesn't (e.g. a stray value for a stale/leftover
		// subscription), just drop the value instead of fetching it live,
		// which would stall delivery of every other incoming message on
		// this connection while it's in flight.
		ep.mu.Lock()
		paramDemand := ep.Parameters[parameter]
		if paramDemand == nil {
			ep.mu.Unlock()
			backend.Logger.Debug("dropping parameter value for unknown demand", "parameter", parameter)
			return nil
		}
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

		// Pushed exactly once per parameter, regardless of how many streams
		// (panels) are watching it - each stream's own cursor determines
		// which pushed values it has and hasn't drained yet (see
		// ParameterStreamDemand and GetAndClearParameterStreamBuffer).
		receivedAt := time.Now()
		paramDemand.Ring.Push(value)
		if ep.ParameterBufferObserver != nil {
			for _, streamDemand := range streamDemands {
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
		// Start at the ring's current write position, not 0, so a newly
		// opened stream only sees values pushed after it started watching,
		// not the parameter's entire pre-existing backlog.
		cursor: demand.Ring.Cursor(),
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

// GetAndClearParameterStreamBuffer drains every value pushed to this
// parameter's shared ring since this stream's own cursor, and advances the
// cursor accordingly. Lock-free: stream.cursor is only ever touched by the
// single RunParameterStream goroutine that owns this path (see
// ParameterStreamDemand's doc comment), and Ring.DrainSince itself needs no
// lock either (see types.Ring).
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

	values, newCursor, dropped := paramDemand.Ring.DrainSince(stream.cursor)
	stream.cursor = newCursor
	if dropped {
		backend.Logger.Warn(
			"parameter stream fell behind its ring buffer capacity; oldest pending values were dropped",
			"parameter", parameter, "path", path, "ringCapacity", ParameterRingCapacity,
		)
	}
	return values
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
		Ring:       types.NewRing[*pvalue.ParameterValue](ParameterRingCapacity),
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
	subscription, err := client.CreateParameterSubscription(ctx, instance, processor, ep.getChannelParameterListener())
	if err != nil {
		return nil, err
	}
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
