package plugin

import (
	"context"
	"sort"
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
// For Graph queries, this fetches each parameter's full historical sample
// range (GetParameterSamplesInProcessorByNames, the same call
// DatasourceGraphFrame makes for a single parameter) and joins them onto
// shared rows using the same carry-forward policy RunParameterStream's live
// ticks use, so a multi-parameter panel backfills its whole time range on
// open exactly like a single-parameter panel already does - not just one
// current-value point per parameter. For SingleValue/DiscreteValue queries,
// where "historical range" doesn't mean the same thing (SingleValue only
// ever wants the current reading; DiscreteValue uses a separate
// ranges/value-mapping API with no equivalent samples endpoint), this
// instead seeds from each parameter's current value only - see
// datasourceMultiParameterCurrentValueFrame.
func DatasourceMultiParameterGraphFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	if q.Type == Graph {
		frame, err := datasourceMultiParameterHistoricalGraphFrame(ctx, endpoint, q)
		if err != nil {
			return nil, err
		}
		if frame != nil {
			return frame, nil
		}
		// No parameter had any history yet: fall through to the
		// current-value seed so the panel isn't left with an empty schema
		// (matching DatasourceGraphFrame's own "no samples yet" fallback
		// isn't an option here since we need *a* row to establish fields).
	}
	return datasourceMultiParameterCurrentValueFrame(ctx, endpoint, q)
}

// datasourceMultiParameterCurrentValueFrame seeds the initial frame from
// each parameter's current value only (one lightweight fetch per
// parameter), reusing reduceParameterBatch - the exact same per-parameter
// reduction RunParameterStream's multi-parameter ticks go through, so the
// resulting field schema (names, order, types) matches every later live
// push. A parameter with no current value yet is left out, mirroring the
// live path's "never reported" carry-forward case, so the first live frame
// that does include it is a pure addition rather than a change to an
// existing field.
func datasourceMultiParameterCurrentValueFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
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
	setMultiParameterUnitsAndThresholds(ctx, endpoint, frame, fields)
	return frame, nil
}

// historicalParameterSeries is one parameter's full sample range, with any
// gap (Yamcs sample with n=0) carry-forward filled already - see
// carryForwardFillSamples. Kept as a plain float64 series (matching what
// Yamcs's sample endpoint always returns: Sample.Avg/Min/Max are `double`
// regardless of the parameter's own engineering type) rather than typed per
// parameter, consistent with DatasourceGraphFrame's existing single-
// parameter historical frame, which already always renders float64 for the
// same reason - proven not to conflict with live ticks' own numeric field
// type (which can itself vary between the raw type and float64 depending on
// whether a given tick's batch got averaged; see reduceParameterBatch),
// since Grafana Live's schema check cares about field count/name/general
// type family, not exact numeric width.
type historicalParameterSeries struct {
	parameter string
	times     []time.Time
	values    []float64
}

// carryForwardFillSamples turns ConvertSampleBufferToFrame's nullable
// []*float64 (which represents a real Yamcs sample gap, n=0, as a null
// entry) into a carry-forward-filled []float64, matching the same policy
// RunParameterStream's live ticks already use for a quiet parameter -
// "same policy for live" per the historical join this feeds into. Any
// leading gap before this parameter's first real sample is dropped rather
// than guessed at.
func carryForwardFillSamples(frame *data.Frame) ([]time.Time, []float64) {
	timeField := frame.Fields[0]
	valueField := frame.Fields[1]
	times := make([]time.Time, 0, timeField.Len())
	values := make([]float64, 0, timeField.Len())

	var last float64
	haveLast := false
	for i := 0; i < timeField.Len(); i++ {
		if v, ok := valueField.At(i).(*float64); ok && v != nil {
			last = *v
			haveLast = true
		}
		if !haveLast {
			continue
		}
		t, _ := timeField.At(i).(time.Time)
		times = append(times, t)
		values = append(values, last)
	}
	return times, values
}

// mergeHistoricalTimestamps returns the sorted union of every series'
// timestamps, deduplicated by exact equality. Each parameter's own sample
// buckets come from the same requested (start, end, maxPoints), so they
// mostly land on identical bucket boundaries in practice - but this doesn't
// assume that, so parameters with genuinely different sampling still merge
// correctly.
func mergeHistoricalTimestamps(series []historicalParameterSeries) []time.Time {
	seen := make(map[int64]struct{})
	var union []time.Time
	for _, s := range series {
		for _, t := range s.times {
			key := t.UnixNano()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			union = append(union, t)
		}
	}
	sort.Slice(union, func(i, j int) bool { return union[i].Before(union[j]) })
	return union
}

// buildHistoricalField projects one series onto the shared union timeline,
// carrying its last known value forward into any row where it has no
// sample of its own - the historical equivalent of a quiet parameter
// keeping its last value on a live tick (buildMultiParameterFrame). A row
// before this series' first real sample backward-fills from that first
// sample instead, so the field has no undefined rows at all: Grafana Live
// row-aligns fields by index within a frame, so leaving early rows at the
// zero value while other parameters already have real data would silently
// mean an incorrect (0.0) history for this parameter, not a visible gap.
func buildHistoricalField(s historicalParameterSeries, unionTimes []time.Time) *data.Field {
	values := make([]float64, len(unionTimes))
	idx := 0
	var last float64
	haveLast := false
	for i, t := range unionTimes {
		for idx < len(s.times) && !s.times[idx].After(t) {
			last = s.values[idx]
			haveLast = true
			idx++
		}
		switch {
		case haveLast:
			values[i] = last
		case len(s.values) > 0:
			values[i] = s.values[0]
		}
	}
	return data.NewField(s.parameter, nil, values)
}

// datasourceMultiParameterHistoricalGraphFrame joins every parameter's full
// historical sample range onto shared rows. Returns a nil frame (not an
// error) when no parameter has any history yet, so the caller can fall back
// to seeding from current values instead of returning an empty-schema
// frame.
func datasourceMultiParameterHistoricalGraphFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}

	start := time.Unix(int64(q.From), 0)
	end := time.Unix(int64(q.To), 0)

	series := make([]historicalParameterSeries, 0, len(q.Parameters))
	for _, parameter := range q.Parameters {
		samples, err := yamcs.GetParameterSamplesInProcessorByNames(ctx, endpoint.GetInstanceName(), endpoint.GetProcessorName(), parameter, start, end, q.MaxPoints)
		if err != nil {
			// No history (or not a sampleable/numeric parameter): genuinely
			// absent, matching the live path's "never reported" case.
			continue
		}
		sampleFrame := tools.ConvertSampleBufferToFrame(samples, parameter, false, false)
		times, values := carryForwardFillSamples(sampleFrame)
		if len(times) == 0 {
			continue
		}
		series = append(series, historicalParameterSeries{parameter: parameter, times: times, values: values})
	}

	if len(series) == 0 {
		return nil, nil
	}

	unionTimes := mergeHistoricalTimestamps(series)
	fields := make([]*data.Field, 0, len(series)+1)
	fields = append(fields, data.NewField("time", nil, unionTimes))
	for _, s := range series {
		fields = append(fields, buildHistoricalField(s, unionTimes))
	}

	frame := data.NewFrame("response", fields...)
	setMultiParameterUnitsAndThresholds(ctx, endpoint, frame, fields[1:])
	return frame, nil
}

// setMultiParameterUnitsAndThresholds applies each single-parameter initial
// frame's unit/alarm-threshold treatment (DatasourceGraphFrame,
// DatasourceSingleValueFrame, DatasourceDiscreteValueFrame) to every field
// in a multi-parameter frame, not just the first one. This only needs to
// happen on this schema-establishing frame - Grafana Live keeps a channel's
// field Config from here for every later live push (data.IncludeDataOnly),
// the same way it already does for a single-parameter query.
func setMultiParameterUnitsAndThresholds(ctx context.Context, endpoint *source.YamcsEndpoint, frame *data.Frame, fields []*data.Field) {
	for _, field := range fields {
		endpoint.SetUnitAndThresholds(ctx, field.Name, frame)
	}
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
