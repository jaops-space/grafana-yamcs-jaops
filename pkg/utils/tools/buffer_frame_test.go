package tools

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func pointer[T any](v T) *T {
	return &v
}

// TestConvertEventsToFrame tests the ConvertEventsToFrame function.
func TestConvertEventsToFrame(t *testing.T) {
	tests := []struct {
		name   string
		events []*events.Event
		want   *data.Frame
	}{
		{
			name:   "Empty events",
			events: []*events.Event{},
			want: data.NewFrame("response",
				data.NewField("time", nil, []time.Time{}),
				data.NewField("message", nil, []string{}),
				data.NewField("severity", nil, []string{}),
			),
		},
		{
			name: "Single event",
			events: []*events.Event{
				{
					Message:        pointer("Test message"),
					Severity:       events.Event_INFO.Enum(),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			want: data.NewFrame("response",
				data.NewField("time", nil, []time.Time{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}),
				data.NewField("message", nil, []string{"Test message"}),
				data.NewField("severity", nil, []string{"INFO"}),
			),
		},
		{
			name: "Multiple events",
			events: []*events.Event{
				{
					Message:        pointer("Info message"),
					Severity:       events.Event_INFO.Enum(),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
				{
					Message:        pointer("Error message"),
					Severity:       events.Event_ERROR.Enum(),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
			},
			want: data.NewFrame("response",
				data.NewField("time", nil, []time.Time{
					time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				}),
				data.NewField("message", nil, []string{"Info message", "Error message"}),
				data.NewField("severity", nil, []string{"INFO", "ERROR"}),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEventsToFrame(tt.events)
			assert.Equal(t, tt.want.Fields[0].Len(), got.Fields[0].Len())
			assert.Equal(t, tt.want.Fields[1].Len(), got.Fields[1].Len())
			assert.Equal(t, tt.want.Fields[2].Len(), got.Fields[2].Len())
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNameHasPrefix tests the nameHasPrefix helper.
func TestNameHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		prefix string
		want   bool
	}{
		{"Exact match", "prefix", "prefix", true},
		{"Has prefix", "prefixabc", "prefix", true},
		{"No prefix", "abc", "prefix", false},
		{"Shorter string", "pre", "prefix", false},
		{"Empty prefix", "abc", "", true},
		{"Empty string", "", "prefix", false},
		{"Empty both", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nameHasPrefix(tt.s, tt.prefix))
		})
	}
}

// TestNameHasSuffix tests the nameHasSuffix helper.
func TestNameHasSuffix(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		suffix string
		want   bool
	}{
		{"Exact match", "suffix", "suffix", true},
		{"Has suffix", "abcsuffix", "suffix", true},
		{"No suffix", "abc", "suffix", false},
		{"Shorter string", "suf", "suffix", false},
		{"Empty suffix", "abc", "", true},
		{"Empty string", "", "suffix", false},
		{"Empty both", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nameHasSuffix(tt.s, tt.suffix))
		})
	}
}

// TestPrepend tests the prepend helper.
func TestPrepend(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		v    int
		want []int
	}{
		{"Empty slice", []int{}, 1, []int{1}},
		{"Non-empty", []int{2, 3}, 1, []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, prepend(tt.s, tt.v))
		})
	}

	// Test with strings
	strTests := []struct {
		name string
		s    []string
		v    string
		want []string
	}{
		{"Empty slice", []string{}, "a", []string{"a"}},
		{"Non-empty", []string{"b", "c"}, "a", []string{"a", "b", "c"}},
	}

	for _, tt := range strTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, prepend(tt.s, tt.v))
		})
	}
}

// TestConvertCommandListToFrame tests the ConvertCommandListToFrame function.
func TestConvertCommandListToFrame(t *testing.T) {
	tests := []struct {
		name     string
		commands []*commanding.CommandHistoryEntry
		wantLen  int
	}{
		{
			name:     "Empty commands",
			commands: []*commanding.CommandHistoryEntry{},
			wantLen:  0,
		},
		{
			name: "Basic command without extras",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd1"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr:           []*commanding.CommandHistoryAttribute{},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with comment and arguments",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd2"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments: []*commanding.CommandAssignment{
						{Name: pointer("arg1"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("val1")}, UserInput: pointer(true)},
					},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: pointer("comment"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("test comment")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with acknowledgements",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd3"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: pointer("Acknowledge_Queued_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("OK")}},
						{Name: pointer("Acknowledge_Queued_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("123")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with extra acknowledgements",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd4"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: pointer("Verifier_Test_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("OK")}},
						{Name: pointer("Verifier_Test_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("456")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with completion",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd5"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: pointer("CommandComplete_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("Completed")}},
						{Name: pointer("CommandComplete_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("789")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Invalid JSON marshal (but skip)",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             pointer("cmd6"),
					CommandName:    pointer("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr:           []*commanding.CommandHistoryAttribute{},
				},
			},
			wantLen: 1, // Assuming no error, as marshal is simple
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertCommandListToFrame(tt.commands)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
			if tt.wantLen > 0 {
				var cmdEntry CommandEntry
				err := json.Unmarshal(got.Fields[0].At(0).(json.RawMessage), &cmdEntry)
				require.NoError(t, err)
				assert.NotEmpty(t, cmdEntry.Id)
			}
		})
	}
}

// TestConvertSampleBufferToFrame tests the ConvertSampleBufferToFrame function.
func TestConvertSampleBufferToFrame(t *testing.T) {
	tests := []struct {
		name       string
		buffer     []*pvalue.TimeSeries_Sample
		parameter  string
		includeMin bool
		includeMax bool
		wantFields int
		wantLen    int
	}{
		{"Empty buffer", []*pvalue.TimeSeries_Sample{}, "param", false, false, 2, 0},
		{"Single sample", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: ptr(1.0), Min: ptr(0.5), Max: ptr(1.5), N: pointer[int32](1)},
		}, "param", true, true, 4, 1},
		{"Multiple samples", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: ptr(1.0), Min: ptr(0.5), Max: ptr(1.5), N: pointer[int32](1)},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), Avg: ptr(2.0), Min: ptr(1.5), Max: ptr(2.5), N: pointer[int32](1)},
		}, "param", false, false, 2, 2},
		{"With nulls", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: ptr(1.0), Min: ptr(0.5), Max: ptr(1.5), N: pointer[int32](1)},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), N: pointer[int32](0)},
		}, "param", true, true, 4, 2}, // Null appended
		{"Consecutive nulls (but logic appends only one if after non-null)", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), N: pointer[int32](0)},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), N: pointer[int32](0)},
			{Time: timestamppb.New(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)), N: pointer[int32](0)},
		}, "param", true, true, 4, 1}, // Only one nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertSampleBufferToFrame(tt.buffer, tt.parameter, tt.includeMin, tt.includeMax)
			assert.Equal(t, tt.wantFields, len(got.Fields))
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
		})
	}
}

// TestConvertSampleBufferToFrameWithOffset tests the ConvertSampleBufferToFrameWithOffset function.
func TestConvertSampleBufferToFrameWithOffset(t *testing.T) {
	offset := time.Hour
	tests := []struct {
		name       string
		buffer     []*pvalue.TimeSeries_Sample
		parameter  string
		includeMin bool
		includeMax bool
		wantLen    int
	}{
		{"Empty", []*pvalue.TimeSeries_Sample{}, "param", false, false, 0},
		{"With offset", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: ptr(1.0), N: pointer[int32](1)},
		}, "param", false, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertSampleBufferToFrameWithOffset(tt.buffer, tt.parameter, tt.includeMin, tt.includeMax, offset)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
			if tt.wantLen > 0 {
				assert.Equal(t, time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), got.Fields[0].At(0).(time.Time))
			}
		})
	}
}

// TestConvertBufferToFrame tests the ConvertBufferToFrame function.
func TestConvertBufferToFrame(t *testing.T) {
	tests := []struct {
		name          string
		buffer        []*pvalue.ParameterValue
		parameter     string
		includeMin    bool
		includeMax    bool
		aggregatePath string
		realtime      bool
		wantLen       int
	}{
		{"Empty", []*pvalue.ParameterValue{}, "param", false, false, "", false, 0},
		{"Empty return default", []*pvalue.ParameterValue{}, "param", false, false, "", false, 0}, // Checks default frame
		{"With values", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
		}, "param", true, true, "", false, 1},
		{"Realtime uses now", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
		}, "param", false, false, "", true, 1},
		{"Aggregate path", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{
				Type: protobuf.Value_AGGREGATE.Enum(),
				AggregateValue: &protobuf.AggregateValue{
					Name:  []string{"x"},
					Value: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
				},
			}},
		}, "param", false, false, ".x", false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertBufferToFrame(tt.buffer, tt.parameter, tt.includeMin, tt.includeMax, tt.aggregatePath, tt.realtime)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
		})
	}
}

// TestConvertRangesToFrame tests the ConvertRangesToFrame function.
func TestConvertRangesToFrame(t *testing.T) {
	tests := []struct {
		name          string
		ranges        *pvalue.Ranges
		parameter     string
		aggregatePath string
		wantLen       int
	}{
		{"Nil ranges", nil, "param", "", 0},
		{"Empty ranges", &pvalue.Ranges{}, "param", "", 0},
		{"With ranges", &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
			{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("val1")}}},
		}}, "param", "", 1},
		{"Aggregate path", &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
			{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{
				Type: protobuf.Value_AGGREGATE.Enum(),
				AggregateValue: &protobuf.AggregateValue{
					Name:  []string{"x"},
					Value: []*protobuf.Value{{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("val1")}},
				},
			}}},
		}}, "param", ".x", 1},
		{"Empty eng values", &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
			{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{}},
		}}, "param", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertRangesToFrame(tt.ranges, tt.parameter, tt.aggregatePath)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
			if tt.wantLen > 0 {
				assert.NotNil(t, got.Fields[1].Config)
			}
		})
	}
}

// TestConvertBufferToAverageFrame tests the ConvertBufferToAverageFrame function.
func TestConvertBufferToAverageFrame(t *testing.T) {
	tests := []struct {
		name          string
		buffer        []*pvalue.ParameterValue
		parameter     string
		getMin        bool
		getMax        bool
		aggregatePath string
		realtime      bool
		wantFields    int
		wantLen       int
	}{
		{"Empty", []*pvalue.ParameterValue{}, "param", false, false, "", false, 1, 0},
		{"Numeric avg", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(3.0)}},
		}, "param", true, true, "", false, 4, 1},
		{"String most frequent", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("a")}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("a")}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("b")}},
		}, "param", false, false, "", false, 2, 1},
		{"Realtime", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
		}, "param", false, false, "", true, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertBufferToAverageFrame(tt.buffer, tt.parameter, tt.getMin, tt.getMax, tt.aggregatePath, tt.realtime)
			assert.Equal(t, tt.wantFields, len(got.Fields))
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
		})
	}
}

// TestExtractParameterValues tests the extractParameterValues function.
func TestExtractParameterValues(t *testing.T) {
	tests := []struct {
		name          string
		buffer        []*pvalue.ParameterValue
		aggregatePath string
		realtime      bool
		wantValuesLen int
		wantTimesLen  int
	}{
		{"Empty", []*pvalue.ParameterValue{}, "", false, 0, 0},
		{"Basic", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
		}, "", false, 1, 1},
		{"Realtime", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.0)}},
		}, "", true, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, times := extractParameterValues(tt.buffer, tt.aggregatePath, tt.realtime)
			assert.Equal(t, tt.wantValuesLen, len(values))
			assert.Equal(t, tt.wantTimesLen, len(times))
		})
	}
}

// TestExtractValue tests the extractValue function.
func TestExtractValue(t *testing.T) {
	tests := []struct {
		name          string
		v             *protobuf.Value
		aggregatePath string
		want          interface{}
	}{
		{"Double", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}, "", 1.5},
		{"Binary", &protobuf.Value{Type: protobuf.Value_BINARY.Enum(), BinaryValue: []byte{0x01, 0x02}}, "", "00000001 00000010"},
		{"Timestamp", &protobuf.Value{Type: protobuf.Value_TIMESTAMP.Enum(), TimestampValue: pointer[int64](123456)}, "", int64(123456)},
		{"Sint64", &protobuf.Value{Type: protobuf.Value_SINT64.Enum(), Sint64Value: pointer[int64](-123)}, "", int64(-123)},
		{"Uint64", &protobuf.Value{Type: protobuf.Value_UINT64.Enum(), Uint64Value: pointer[uint64](123)}, "", uint64(123)},
		{"Sint32", &protobuf.Value{Type: protobuf.Value_SINT32.Enum(), Sint32Value: pointer[int32](-123)}, "", int32(-123)},
		{"Uint32", &protobuf.Value{Type: protobuf.Value_UINT32.Enum(), Uint32Value: pointer[uint32](123)}, "", uint32(123)},
		{"Float", &protobuf.Value{Type: protobuf.Value_FLOAT.Enum(), FloatValue: pointer[float32](1.5)}, "", 1.5},
		{"Boolean true", &protobuf.Value{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: pointer(true)}, "", "true"},
		{"Boolean false", &protobuf.Value{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: pointer(false)}, "", "false"},
		{"String", &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: pointer("test")}, "", "test"},
		{"Aggregate with path", &protobuf.Value{
			Type: protobuf.Value_AGGREGATE.Enum(),
			AggregateValue: &protobuf.AggregateValue{
				Name:  []string{"x"},
				Value: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
			},
		}, ".x", 1.5},
		{"Aggregate no path", &protobuf.Value{Type: protobuf.Value_AGGREGATE.Enum()}, "", int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractValue(tt.v, tt.aggregatePath))
		})
	}
}

// TestSplitPath tests the splitPath function.
func TestSplitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{"Empty", "", []string(nil)},
		{"Simple", ".x", []string{"x"}},
		{"Array", "[0]", []string{"[0]"}},
		{"Complex", ".x[0].y[2]", []string{"x", "[0]", "y", "[2]"}},
		{"No dot", "x[0]", []string{"x", "[0]"}},
		{"Invalid chars", ".x.y", []string{"x", "y"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitPath(tt.path))
		})
	}
}

// TestAggregateExtractFromPath tests the aggregateExtractFromPath function.
func TestAggregateExtractFromPath(t *testing.T) {
	defaultValue := &protobuf.Value{Type: protobuf.Value_SINT64.Enum(), Sint64Value: new(int64)}

	tests := []struct {
		name  string
		value *protobuf.Value
		path  string
		want  *protobuf.Value
	}{
		{"Empty path", &protobuf.Value{}, "", defaultValue},
		{"Invalid path", &protobuf.Value{Type: protobuf.Value_AGGREGATE.Enum()}, ".invalid", defaultValue},
		{"Aggregate valid", &protobuf.Value{
			Type: protobuf.Value_AGGREGATE.Enum(),
			AggregateValue: &protobuf.AggregateValue{
				Name:  []string{"x"},
				Value: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
			},
		}, ".x", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
		{"Array valid", &protobuf.Value{
			Type:       protobuf.Value_ARRAY.Enum(),
			ArrayValue: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
		}, "[0]", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
		{"Array out of bounds", &protobuf.Value{
			Type:       protobuf.Value_ARRAY.Enum(),
			ArrayValue: []*protobuf.Value{},
		}, "[0]", defaultValue},
		{"Nested", &protobuf.Value{
			Type: protobuf.Value_AGGREGATE.Enum(),
			AggregateValue: &protobuf.AggregateValue{
				Name: []string{"x"},
				Value: []*protobuf.Value{{
					Type:       protobuf.Value_ARRAY.Enum(),
					ArrayValue: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
				}},
			},
		}, ".x[0]", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
		{"Case insensitive name", &protobuf.Value{
			Type: protobuf.Value_AGGREGATE.Enum(),
			AggregateValue: &protobuf.AggregateValue{
				Name:  []string{"X"},
				Value: []*protobuf.Value{{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
			},
		}, ".x", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: pointer(1.5)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregateExtractFromPath(tt.value, tt.path)
			assert.Equal(t, tt.want.GetType(), got.GetType())
			assert.Equal(t, tt.want.GetDoubleValue(), got.GetDoubleValue())
		})
	}
}

// TestFormatBinary tests the formatBinary function.
func TestFormatBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"Empty", []byte{}, ""},
		{"Single byte", []byte{0x01}, "00000001"},
		{"Multiple", []byte{0x01, 0xFF}, "00000001 11111111"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBinary(tt.data))
		})
	}
}

// TestCreateValueField tests the CreateValueField function.
func TestCreateValueField(t *testing.T) {
	tests := []struct {
		name     string
		values   []interface{}
		param    string
		wantType reflect.Type
	}{
		{"Int64", []interface{}{int64(1)}, "param", reflect.TypeFor[int64]()},
		{"Uint64", []interface{}{uint64(1)}, "param", reflect.TypeFor[uint64]()},
		{"Int32", []interface{}{int32(1)}, "param", reflect.TypeFor[int32]()},
		{"Uint32", []interface{}{uint32(1)}, "param", reflect.TypeFor[uint32]()},
		{"Float64", []interface{}{float64(1.0)}, "param", reflect.TypeFor[float64]()},
		{"Bool", []interface{}{true}, "param", reflect.TypeFor[bool]()},
		{"String", []interface{}{"test"}, "param", reflect.TypeFor[string]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateValueField(tt.values, tt.param)
			assert.Equal(t, tt.wantType, reflect.TypeOf(got.At(0)))
		})
	}
}

// TestConvertSlice tests the ConvertSlice function.
func TestConvertSlice(t *testing.T) {
	tests := []struct {
		name   string
		values []interface{}
		want   []int
	}{
		{"Empty", []interface{}{}, []int{}},
		{"Ints", []interface{}{1, 2}, []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ConvertSlice[int](tt.values))
		})
	}

	// Test panic on wrong type
	assert.Panics(t, func() { ConvertSlice[int]([]interface{}{"a"}) })
}

// TestConvert tests the convert function (similar to ConvertSlice).
func TestConvert(t *testing.T) {
	tests := []struct {
		name   string
		values []interface{}
		want   []float64
	}{
		{"Empty", []interface{}{}, []float64{}},
		{"Floats", []interface{}{1.0, 2.0}, []float64{1.0, 2.0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convert[float64](tt.values))
		})
	}
}

// TestCalculateStats tests the CalculateStats function.
func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name      string
		values    []interface{}
		parameter string
		wantAvg   float64
		wantMin   interface{}
		wantMax   interface{}
	}{
		{"Empty", []interface{}{}, "param", 0, nil, nil},
		{"Int64", []interface{}{int64(1), int64(3)}, "param", 2.0, int64(1), int64(3)},
		{"Uint64", []interface{}{uint64(1), uint64(3)}, "param", 2.0, uint64(1), uint64(3)},
		{"Int32", []interface{}{int32(1), int32(3)}, "param", 2.0, int32(1), int32(3)},
		{"Uint32", []interface{}{uint32(1), uint32(3)}, "param", 2.0, uint32(1), uint32(3)},
		{"Float64", []interface{}{1.0, 3.0}, "param", 2.0, 1.0, 3.0},
		{"String most freq", []interface{}{"a", "a", "b"}, "param", 0, nil, nil}, // Avg is most freq "a"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avg, min, max := CalculateStats(tt.values, tt.parameter)
			if len(tt.values) > 0 && reflect.TypeOf(tt.values[0]).Kind() != reflect.String {
				assert.Equal(t, tt.wantAvg, avg.At(0).(float64))
				if min != nil {
					assert.Equal(t, tt.wantMin, min.At(0))
				}
				if max != nil {
					assert.Equal(t, tt.wantMax, max.At(0))
				}
			} else if len(tt.values) > 0 && reflect.TypeOf(tt.values[0]).Kind() == reflect.String {
				assert.Equal(t, "a", avg.At(0).(string))
			}
		})
	}
}

// TestCreateStatFields tests the createStatFields function.
func TestCreateStatFields(t *testing.T) {
	tests := []struct {
		name   string
		param  string
		values []int32
		sum    int32
		min    int32
		max    int32
	}{
		{"Basic", "param", []int32{1, 3}, 4, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avg, minF, maxF := createStatFields(tt.param, tt.values, tt.sum, tt.min, tt.max)
			assert.Equal(t, 2.0, avg.At(0).(float64))
			assert.Equal(t, tt.min, minF.At(0).(int32))
			assert.Equal(t, tt.max, maxF.At(0).(int32))
		})
	}
}

// TestSum tests the Sum function.
func TestSum(t *testing.T) {
	tests := []struct {
		name   string
		values []int
		want   int
	}{
		{"Empty", []int{}, 0},
		{"Positive", []int{1, 2, 3}, 6},
		{"Negative", []int{-1, -2}, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Sum(tt.values))
		})
	}

	// Float
	floatTests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"Floats", []float64{1.5, 2.5}, 4.0},
	}
	for _, tt := range floatTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Sum(tt.values))
		})
	}
}

// TestMinMax tests the MinMax function.
func TestMinMax(t *testing.T) {
	tests := []struct {
		name    string
		values  []int
		wantMin int
		wantMax int
	}{
		{"Single", []int{5}, 5, 5},
		{"Multiple", []int{1, 3, 2}, 1, 3},
		{"Negative", []int{-1, -3}, -3, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := MinMax(tt.values)
			assert.Equal(t, tt.wantMin, min)
			assert.Equal(t, tt.wantMax, max)
		})
	}
}

// TestMostFrequent tests the MostFrequent function.
func TestMostFrequent(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"Empty", []string{}, ""},
		{"Single", []string{"a"}, "a"},
		{"Multiple same", []string{"a", "a"}, "a"},
		{"Tie picks first", []string{"a", "b", "a", "b"}, "a"}, // Picks the first encountered
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.values) == 0 {
				assert.Equal(t, "", MostFrequent(tt.values))
			} else {
				assert.Equal(t, tt.want, MostFrequent(tt.values))
			}
		})
	}
}

// TestHashString tests the hashString function.
func TestHashString(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{"Empty", ""},
		{"Test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deterministic, but no specific want, just check it's int
			assert.IsType(t, 0, hashString(tt.s))
		})
	}
}

// TestHashToRGB tests the HashToRGB function.
func TestHashToRGB(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{"Test", "test"},
		{"Empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashToRGB(tt.s)
			assert.Regexp(t, "^#[0-9A-F]{6}$", got)
		})
	}
}

// Helper to create float64 pointer
func ptr(f float64) *float64 {
	return &f
}
