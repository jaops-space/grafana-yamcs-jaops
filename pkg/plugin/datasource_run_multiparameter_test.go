package plugin

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func numericValue(t *testing.T, seconds int64, value float64) *pvalue.ParameterValue {
	t.Helper()
	status := pvalue.AcquisitionStatus_ACQUIRED
	return &pvalue.ParameterValue{
		GenerationTime:    timestamppb.New(time.Unix(seconds, 0)),
		AcquisitionStatus: &status,
		EngValue: &protobuf.Value{
			Type:        protobuf.Value_DOUBLE.Enum(),
			DoubleValue: &value,
		},
	}
}

func expiredValue(t *testing.T, seconds int64, value float64) *pvalue.ParameterValue {
	t.Helper()
	v := numericValue(t, seconds, value)
	status := pvalue.AcquisitionStatus_EXPIRED
	v.AcquisitionStatus = &status
	return v
}

func enumValue(t *testing.T, seconds int64, state string) *pvalue.ParameterValue {
	t.Helper()
	status := pvalue.AcquisitionStatus_ACQUIRED
	return &pvalue.ParameterValue{
		GenerationTime:    timestamppb.New(time.Unix(seconds, 0)),
		AcquisitionStatus: &status,
		EngValue: &protobuf.Value{
			Type:        protobuf.Value_ENUMERATED.Enum(),
			StringValue: &state,
		},
	}
}

func newStates(parameters ...string) map[string]*multiParameterState {
	states := make(map[string]*multiParameterState, len(parameters))
	for _, parameter := range parameters {
		states[parameter] = &multiParameterState{}
	}
	return states
}

func TestBuildMultiParameterFrameJoinsFreshValuesFromAllParameters(t *testing.T) {
	parameters := []string{"/a", "/b"}
	states := newStates(parameters...)
	batches := map[string][]*pvalue.ParameterValue{
		"/a": {numericValue(t, 100, 1.5)},
		"/b": {numericValue(t, 100, 2.5)},
	}

	frame, totalValues, anyFresh := buildMultiParameterFrame(Graph, false, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return batches[parameter]
	})

	if !anyFresh {
		t.Fatalf("expected anyFresh=true")
	}
	if totalValues != 2 {
		t.Fatalf("expected totalValues=2, got %d", totalValues)
	}
	if frame == nil || len(frame.Fields) != 3 { // time + 2 parameters
		t.Fatalf("expected a joined frame with 3 fields, got %#v", frame)
	}
	if frame.Fields[1].At(0) != 1.5 || frame.Fields[2].At(0) != 2.5 {
		t.Fatalf("expected joined values 1.5 and 2.5, got %v and %v", frame.Fields[1].At(0), frame.Fields[2].At(0))
	}
}

func TestBuildMultiParameterFrameCarriesForwardQuietParameters(t *testing.T) {
	parameters := []string{"/a", "/b"}
	states := newStates(parameters...)

	// Tick 1: both parameters report.
	batches := map[string][]*pvalue.ParameterValue{
		"/a": {numericValue(t, 100, 1.5)},
		"/b": {numericValue(t, 100, 2.5)},
	}
	drain := func(parameter string) []*pvalue.ParameterValue { return batches[parameter] }
	if _, _, anyFresh := buildMultiParameterFrame(Graph, false, parameters, states, drain); !anyFresh {
		t.Fatalf("expected first tick to be fresh")
	}

	// Tick 2: only /a reports; /b should carry forward its last value (2.5),
	// not disappear from the frame or go null.
	batches = map[string][]*pvalue.ParameterValue{
		"/a": {numericValue(t, 101, 9.0)},
		"/b": nil,
	}
	frame, totalValues, anyFresh := buildMultiParameterFrame(Graph, false, parameters, states, drain)

	if !anyFresh {
		t.Fatalf("expected anyFresh=true when at least one parameter has new data")
	}
	if totalValues != 1 {
		t.Fatalf("expected totalValues=1 (only /a produced new samples), got %d", totalValues)
	}
	if len(frame.Fields) != 3 {
		t.Fatalf("expected /b to still be present (carried forward), got %d fields", len(frame.Fields))
	}
	if frame.Fields[1].At(0) != 9.0 {
		t.Fatalf("expected /a's fresh value 9.0, got %v", frame.Fields[1].At(0))
	}
	if frame.Fields[2].At(0) != 2.5 {
		t.Fatalf("expected /b's carried-forward value 2.5, got %v", frame.Fields[2].At(0))
	}
}

func TestBuildMultiParameterFrameOmitsParameterThatHasNeverReported(t *testing.T) {
	parameters := []string{"/a", "/b"}
	states := newStates(parameters...)
	batches := map[string][]*pvalue.ParameterValue{
		"/a": {numericValue(t, 100, 1.5)},
		"/b": nil, // /b has never sent a single value yet
	}

	frame, _, anyFresh := buildMultiParameterFrame(Graph, false, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return batches[parameter]
	})

	if !anyFresh {
		t.Fatalf("expected anyFresh=true")
	}
	if len(frame.Fields) != 2 { // time + /a only, /b genuinely absent
		t.Fatalf("expected /b to be absent (never reported), got %d fields", len(frame.Fields))
	}
	if frame.Fields[1].Name != "/a" {
		t.Fatalf("expected the present field to be /a, got %q", frame.Fields[1].Name)
	}
}

func TestBuildMultiParameterFrameSendsNothingWhenAllParametersAreQuiet(t *testing.T) {
	parameters := []string{"/a", "/b"}
	states := newStates(parameters...)
	// Seed both with an initial value first.
	drain := func(parameter string) []*pvalue.ParameterValue {
		return []*pvalue.ParameterValue{numericValue(t, 100, 1.0)}
	}
	buildMultiParameterFrame(Graph, false, parameters, states, drain)

	// Now a tick where nothing new arrived for either parameter.
	frame, totalValues, anyFresh := buildMultiParameterFrame(Graph, false, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return nil
	})

	if anyFresh {
		t.Fatalf("expected anyFresh=false when no parameter produced new data")
	}
	if frame != nil {
		t.Fatalf("expected no frame to be built on an all-quiet tick, got %#v", frame)
	}
	if totalValues != 0 {
		t.Fatalf("expected totalValues=0, got %d", totalValues)
	}
}

func TestBuildMultiParameterFrameUsesLatestGenerationTimeNotWallClock(t *testing.T) {
	parameters := []string{"/a", "/b"}
	states := newStates(parameters...)

	// /a's sample is generation-timestamped well before /b's - simulating a
	// replay where these timestamps have nothing to do with wall-clock time.
	replayEpoch := int64(946684800) // 2000-01-01, deliberately not "now"
	batches := map[string][]*pvalue.ParameterValue{
		"/a": {numericValue(t, replayEpoch, 1.0)},
		"/b": {numericValue(t, replayEpoch+5, 2.0)},
	}

	before := time.Now()
	frame, _, _ := buildMultiParameterFrame(Graph, false, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return batches[parameter]
	})
	after := time.Now()

	rowTime, ok := frame.Fields[0].At(0).(time.Time)
	if !ok {
		t.Fatalf("expected time field to hold a time.Time")
	}
	wantRowTime := time.Unix(replayEpoch+5, 0)
	if !rowTime.Equal(wantRowTime) {
		t.Fatalf("expected row time to be the latest GenerationTime (%v), got %v", wantRowTime, rowTime)
	}
	if (rowTime.After(before) || rowTime.Equal(before)) && rowTime.Before(after) {
		t.Fatalf("row time landed in the wall-clock window of the test run - it should reflect replay time, not time.Now()")
	}
}

func TestBuildMultiParameterFrameReportsExpiredNoticeFromYamcsAcquisitionStatus(t *testing.T) {
	parameters := []string{"/a"}
	states := newStates(parameters...)
	batches := []*pvalue.ParameterValue{expiredValue(t, 100, 1.0)}

	frame, _, _ := buildMultiParameterFrame(Graph, false, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return batches
	})

	if frame.Meta == nil || len(frame.Meta.Notices) == 0 {
		t.Fatalf("expected an expired-parameter notice, got none")
	}
}

func TestBuildMultiParameterFrameUsesDiscreteConversionAndKeepsColorMappingOnCarryForward(t *testing.T) {
	// Regression test: a batched DiscreteValue query must get the same
	// value-mapped/colored conversion (ConvertDiscreteBufferToFrame) a
	// single-parameter discrete panel gets, not the plain numeric Graph
	// conversion - and that mapping must survive onto a carried-forward tick
	// too, not just the tick that produced it.
	parameters := []string{"/mode"}
	states := newStates(parameters...)

	batches := []*pvalue.ParameterValue{enumValue(t, 100, "ARMED")}
	drain := func(parameter string) []*pvalue.ParameterValue { return batches }
	frame, _, _ := buildMultiParameterFrame(DiscreteValue, true, parameters, states, drain)

	if frame.Fields[1].At(0) != "ARMED" {
		t.Fatalf("expected discrete value %q, got %v", "ARMED", frame.Fields[1].At(0))
	}
	if frame.Fields[1].Config == nil || len(frame.Fields[1].Config.Mappings) == 0 {
		t.Fatalf("expected a value mapping on the fresh tick's field, got none")
	}

	// Now a quiet tick: /mode carries forward, and must keep its mapping.
	frame, _, anyFresh := buildMultiParameterFrame(DiscreteValue, true, parameters, states, func(parameter string) []*pvalue.ParameterValue {
		return nil
	})
	if anyFresh {
		t.Fatalf("expected the quiet tick to report anyFresh=false")
	}
	if frame != nil {
		t.Fatalf("expected no frame on an all-quiet tick even for a carried-forward parameter")
	}

	// A carried-forward value is only visible once another parameter is
	// fresh in the same tick (anyFresh requires at least one fresh
	// parameter) - simulate that directly via the state instead.
	if states["/mode"].lastField.Config == nil || len(states["/mode"].lastField.Config.Mappings) == 0 {
		t.Fatalf("expected the carried-forward field to keep its value mapping")
	}
}

func sampleFrame(t *testing.T, times []time.Time, values []*float64) *data.Frame {
	t.Helper()
	timeField := data.NewField("time", nil, times)
	valueField := data.NewField("/a", nil, values)
	return data.NewFrame("response", timeField, valueField)
}

func floatPtr(v float64) *float64 { return &v }

func TestCarryForwardFillSamplesDropsLeadingGapAndFillsInteriorGaps(t *testing.T) {
	base := time.Unix(1000, 0)
	times := []time.Time{base, base.Add(time.Second), base.Add(2 * time.Second), base.Add(3 * time.Second)}
	// A leading gap (no value yet), then a real value, then a gap (carried
	// forward), then a new real value.
	values := []*float64{nil, floatPtr(1.0), nil, floatPtr(2.0)}

	gotTimes, gotValues := carryForwardFillSamples(sampleFrame(t, times, values))

	if len(gotTimes) != 3 {
		t.Fatalf("expected the leading gap dropped (3 rows), got %d: %v", len(gotTimes), gotValues)
	}
	if gotValues[0] != 1.0 || gotValues[1] != 1.0 || gotValues[2] != 2.0 {
		t.Fatalf("expected [1.0 1.0 2.0] (interior gap carried forward), got %v", gotValues)
	}
}

func TestCarryForwardFillSamplesAllGapsProducesEmpty(t *testing.T) {
	base := time.Unix(1000, 0)
	times := []time.Time{base, base.Add(time.Second)}
	values := []*float64{nil, nil}

	gotTimes, gotValues := carryForwardFillSamples(sampleFrame(t, times, values))

	if len(gotTimes) != 0 || len(gotValues) != 0 {
		t.Fatalf("expected no rows when a parameter never had a real sample, got %v %v", gotTimes, gotValues)
	}
}

func TestMergeHistoricalTimestampsDeduplicatesAndSorts(t *testing.T) {
	base := time.Unix(1000, 0)
	a := historicalParameterSeries{parameter: "/a", times: []time.Time{base, base.Add(2 * time.Second)}}
	b := historicalParameterSeries{parameter: "/b", times: []time.Time{base.Add(time.Second), base.Add(2 * time.Second)}}

	union := mergeHistoricalTimestamps([]historicalParameterSeries{a, b})

	if len(union) != 3 {
		t.Fatalf("expected 3 distinct timestamps (one shared), got %d: %v", len(union), union)
	}
	for i := 1; i < len(union); i++ {
		if union[i].Before(union[i-1]) {
			t.Fatalf("expected ascending order, got %v", union)
		}
	}
}

func TestBuildHistoricalFieldCarriesForwardAndBackfillsBeforeFirstSample(t *testing.T) {
	base := time.Unix(1000, 0)
	// /a has a sample at t=1 and t=3; the union also has rows at t=0 and t=2
	// where /a has no sample of its own.
	series := historicalParameterSeries{
		parameter: "/a",
		times:     []time.Time{base.Add(time.Second), base.Add(3 * time.Second)},
		values:    []float64{10, 30},
	}
	union := []time.Time{base, base.Add(time.Second), base.Add(2 * time.Second), base.Add(3 * time.Second)}

	field := buildHistoricalField(series, union)

	got := []float64{
		field.At(0).(float64), field.At(1).(float64), field.At(2).(float64), field.At(3).(float64),
	}
	want := []float64{10, 10, 10, 30}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v (t=0 backfilled from first sample, t=2 carried forward from t=1), got %v", want, got)
		}
	}
}

func TestDatasourceMultiParameterHistoricalGraphFrameJoinsAlignedAndMisalignedParameters(t *testing.T) {
	base := time.Unix(1000, 0)
	// /a and /b share a timestamp (the common case: same request window/
	// maxPoints bucket the same way); /c only has a later, unique sample.
	a := historicalParameterSeries{parameter: "/a", times: []time.Time{base, base.Add(time.Second)}, values: []float64{1, 2}}
	b := historicalParameterSeries{parameter: "/b", times: []time.Time{base, base.Add(time.Second)}, values: []float64{10, 20}}
	c := historicalParameterSeries{parameter: "/c", times: []time.Time{base.Add(2 * time.Second)}, values: []float64{100}}

	union := mergeHistoricalTimestamps([]historicalParameterSeries{a, b, c})
	if len(union) != 3 {
		t.Fatalf("expected 3 rows (2 shared + 1 unique), got %d: %v", len(union), union)
	}

	fields := []*data.Field{data.NewField("time", nil, union)}
	for _, s := range []historicalParameterSeries{a, b, c} {
		fields = append(fields, buildHistoricalField(s, union))
	}
	frame := data.NewFrame("response", fields...)

	if len(frame.Fields) != 4 {
		t.Fatalf("expected time + 3 parameter fields, got %d", len(frame.Fields))
	}
	// Row 2 (/c's own sample time): /a and /b carry forward their last
	// known value instead of going undefined.
	if frame.Fields[1].At(2).(float64) != 2 || frame.Fields[2].At(2).(float64) != 20 {
		t.Fatalf("expected /a and /b to carry forward into /c's row, got a=%v b=%v",
			frame.Fields[1].At(2), frame.Fields[2].At(2))
	}
	if frame.Fields[3].At(2).(float64) != 100 {
		t.Fatalf("expected /c's own value 100 at its row, got %v", frame.Fields[3].At(2))
	}
}
