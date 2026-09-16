package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

// multiParameterState is the carry-forward state RunParameterStream's
// multi-parameter path keeps per parameter, across ticks, for the lifetime
// of one stream (i.e. one open panel). It lets a parameter that didn't
// report new data on a given tick still contribute its last known value to
// that tick's row, instead of the row's field count shrinking and growing
// as parameters go quiet and noisy independently of each other.
type multiParameterState struct {
	// lastField holds the parameter's most recently produced value, as a
	// length-1 data.Field of whatever concrete type that parameter is
	// (float64, string, nullable variants, ...). Built once when fresh data
	// arrives via singleValueField, then reused verbatim on every quiet tick
	// until new data replaces it. nil until the parameter has reported at
	// least once.
	lastField *data.Field
	// lastGenerationTime is that same value's own Yamcs-reported
	// GenerationTime (never local wall-clock time - see RunParameterStream's
	// doc comment for why). Used to pick the row's shared timestamp.
	lastGenerationTime time.Time
	// lastRaw is the raw protobuf value backing lastField, kept only so a
	// quiet tick can still re-check/re-report Yamcs's own expiration status
	// via tools.AppendExpiredParameterNotice without storing that logic
	// twice.
	lastRaw *pvalue.ParameterValue
}

// multiParameterStreamPath namespaces the per-parameter ring buffer demand
// registered with the endpoint under this stream's Grafana Live channel
// path, so two different queries that happen to share a parameter don't
// collide on the same demand key. Also used (harmlessly) for the
// single-parameter case, for one consistent naming scheme either way.
func multiParameterStreamPath(basePath string, parameter string) string {
	return basePath + "/" + parameter
}

// convertParameterBatchByType converts one parameter's batch into a frame
// exactly the way the single-parameter path always has, dispatching on
// query type: DiscreteValue gets value-mapped colors/labels
// (ConvertDiscreteBufferToFrame), SingleValue gets the un-averaged
// current-value framing (ConvertSingleValueBufferToFrame), and
// Graph/Image average bursts of more than 3 samples
// (ConvertBufferToAverageFrame) or pass them through as-is otherwise
// (ConvertBufferToFrame). Shared by RunParameterStream's single-parameter
// tick handler and reduceParameterBatch, so a parameter's own column
// behaves identically whether it's viewed alone or as part of a
// multi-parameter query.
func convertParameterBatchByType(queryType PluginQueryType, batch []*pvalue.ParameterValue, parameter string, automaticColors, getMin, getMax bool) *data.Frame {
	switch queryType {
	case DiscreteValue:
		return tools.ConvertDiscreteBufferToFrame(batch, parameter, automaticColors, false)
	case SingleValue:
		return tools.ConvertSingleValueBufferToFrame(batch, parameter, false)
	default: // Graph, Image
		if len(batch) > 3 {
			return tools.ConvertBufferToAverageFrame(batch, parameter, getMin, getMax, false)
		}
		return tools.ConvertBufferToFrame(batch, parameter, getMin, getMax, false)
	}
}

// singleValueField extracts the value at idx from field as a fresh,
// length-1 field of the exact same underlying type and config (numeric,
// string, nullable, value-mapped/colored, ...), generically - i.e. without
// a type switch over every Yamcs/Grafana field type. This is what makes
// carry-forward possible: the resulting field can be re-appended into next
// tick's joined frame verbatim even though the caller has no idea what
// concrete Go type, or value-mapping config, any given parameter's value
// field carries (e.g. DiscreteValue's color/label mapping).
func singleValueField(field *data.Field, idx int, name string) *data.Field {
	out := data.NewFieldFromFieldType(field.Type(), 0)
	out.Name = name
	out.Config = field.Config
	out.Append(field.At(idx))
	return out
}

// reduceParameterBatch turns a batch of newly-drained values for one
// parameter into the same single representative value RunParameterStream's
// single-parameter path would report for it (see convertParameterBatchByType),
// then collapses that down to just its last row: even a DiscreteValue or
// Graph conversion that returns multiple rows for a bursty tick is reduced
// to one representative value here, since a multi-parameter query's rows
// have to align across parameters (see buildMultiParameterFrame) - there's
// no such alignment constraint for a single parameter, which is why
// RunParameterStream's single-parameter path sends the un-reduced batch
// instead of calling this.
func reduceParameterBatch(queryType PluginQueryType, automaticColors bool, batch []*pvalue.ParameterValue, parameter string) (value *data.Field, generationTime time.Time) {
	frame := convertParameterBatchByType(queryType, batch, parameter, automaticColors, false, false)
	lastIdx := frame.Fields[1].Len() - 1
	return singleValueField(frame.Fields[1], lastIdx, parameter), frame.Fields[0].At(lastIdx).(time.Time)
}

// DatasourceMultiParameterGraphFrame builds the initial frame SubscribeStream
// returns for a multi-parameter Graph/SingleValue/DiscreteValue query.
//
// This has to produce exactly the same schema (field names, order, and
// types) as RunParameterStream's live pushes for that same query, not just
// "some" initial data: Grafana Live's client-side StreamingDataFrame locks
// in a channel's field schema from this first frame and compares every
// later push against it. A first frame that only covered
// q.Parameters[0] (as plain DatasourceGraphFrame would give) has a
// different field count than every subsequent multi-field push, so every
// single live tick for the rest of the panel's life would fail that schema
// check - not just during warmup.
//
// To guarantee that match, this reuses reduceParameterBatch - the exact
// same per-parameter reduction RunParameterStream's multi-parameter ticks
// go through, including the query-type dispatch (DiscreteValue's
// color/label mapping, SingleValue's framing) - seeded from each
// parameter's current value (one lightweight fetch per parameter) rather
// than a full historical range, and joins them with the same "latest wins"
// row-timestamp policy. A parameter with no current value yet is left out,
// mirroring the live path's "never reported" carry-forward case, so the
// first live frame that does include it is a pure addition rather than a
// change to an existing field.
//
// Known limitation: only the parameter's *current* value is fetched, not
// its historical range - a multi-parameter panel's initial render shows one
// data point per parameter until live ticks fill in more, unlike a
// single-parameter panel's full historical backfill (DatasourceGraphFrame).
// Joining full historical ranges across parameters with this same policy is
// a separate problem this doesn't attempt.
func DatasourceMultiParameterGraphFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}

	var rowTime time.Time
	fields := make([]*data.Field, 0, len(q.Parameters))
	for _, parameter := range q.Parameters {
		value, err := yamcs.GetParameterValueByName(ctx, endpoint.GetInstanceName(), endpoint.GetProcessorName(), parameter)
		if err != nil {
			// No current value yet: genuinely absent, matching the live
			// path's "never reported" carry-forward case.
			continue
		}

		field, generationTime := reduceParameterBatch(q.Type, q.AutomaticColors, []*pvalue.ParameterValue{value}, parameter)
		fields = append(fields, field)
		if generationTime.After(rowTime) {
			rowTime = generationTime
		}
	}

	if len(fields) == 0 {
		return data.NewFrame("response", data.NewField("time", nil, []time.Time{})), nil
	}

	timeField := data.NewField("time", nil, []time.Time{rowTime})
	frame := data.NewFrame("response", append([]*data.Field{timeField}, fields...)...)

	// Every single-parameter initial frame (DatasourceGraphFrame,
	// DatasourceSingleValueFrame, DatasourceDiscreteValueFrame) sets each
	// value field's unit and alarm thresholds here; a multi-parameter query
	// needs the same treatment per parameter, not just the first one. This
	// only needs to happen on this schema-establishing frame - Grafana Live
	// keeps a channel's field Config from here for every later live push
	// (data.IncludeDataOnly), the same way it already does for a
	// single-parameter query.
	for _, field := range fields {
		endpoint.SetUnitAndThresholds(ctx, field.Name, frame)
	}

	return frame, nil
}

// buildMultiParameterFrame drains one tick's worth of data for every
// parameter (via drain, decoupled from any real endpoint/network so this
// stays unit-testable), updates each parameter's carry-forward state, and
// joins the result into a single frame using the "latest wins" row-
// timestamp policy:
//
// Row timestamp ("latest wins", not wall clock): each row's timestamp is
// the latest Yamcs-reported GenerationTime among whatever parameters
// produced fresh data this tick - never the local ticker's wall-clock time.
// RunParameterStream's single-parameter path already makes this choice per
// parameter (convertParameterBatchByType passes realtime=false, so frame
// timestamps come from the value's own GenerationTime); this just extends
// the same rule across parameters instead of inventing a new one. That
// matters for replay/simulation, where GenerationTime reflects the
// replayed timeline rather than real time - a row stamped with time.Now()
// would be wrong (and misleading on a time axis) whenever the processor
// isn't running at 1x realtime.
//
// Carry-forward: a parameter with no new data this tick keeps contributing
// its last known value (see multiParameterState) rather than disappearing
// from the row, so a table/state-timeline panel doesn't get a column that
// flickers in and out just because that one parameter happens to update
// less often than the others. A parameter that has never reported at all
// yet is the one case that's genuinely absent from the frame (not
// null-padded) until its first value arrives; after that it's present in
// every row for the life of the stream, real or carried forward. A tick
// where *no* parameter produced fresh data sends nothing, matching the
// single-parameter path's behavior of staying silent on an empty batch.
func buildMultiParameterFrame(
	queryType PluginQueryType,
	automaticColors bool,
	parameters []string,
	states map[string]*multiParameterState,
	drain func(parameter string) []*pvalue.ParameterValue,
) (frame *data.Frame, totalValues int, anyFresh bool) {
	var rowTime time.Time
	fields := make([]*data.Field, 0, len(parameters))

	for _, parameter := range parameters {
		state := states[parameter]
		batch := drain(parameter)
		if len(batch) > 0 {
			anyFresh = true
			totalValues += len(batch)
			state.lastField, state.lastGenerationTime = reduceParameterBatch(queryType, automaticColors, batch, parameter)
			state.lastRaw = batch[len(batch)-1]
		}

		if state.lastField == nil {
			// Never reported at all yet: genuinely absent from the frame,
			// not null-padded, per the carry-forward policy above.
			continue
		}

		fields = append(fields, state.lastField)
		if state.lastGenerationTime.After(rowTime) {
			rowTime = state.lastGenerationTime
		}
	}

	if !anyFresh || len(fields) == 0 {
		return nil, totalValues, anyFresh
	}

	timeField := data.NewField("time", nil, []time.Time{rowTime})
	frame = data.NewFrame("response", append([]*data.Field{timeField}, fields...)...)

	// Reusing Yamcs's own expiration status (rather than a locally-invented
	// staleness timeout) per parameter, exactly like the single-parameter
	// SingleValue frame already does via the same helper.
	for _, parameter := range parameters {
		state := states[parameter]
		if state.lastRaw == nil {
			continue
		}
		tools.AppendExpiredParameterNotice(frame, []*pvalue.ParameterValue{state.lastRaw}, parameter)
	}

	return frame, totalValues, anyFresh
}
