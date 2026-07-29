package tools

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func new[T any](v T) *T {
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
					Message:        new("Test message"),
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
					Message:        new("Info message"),
					Severity:       events.Event_INFO.Enum(),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
				{
					Message:        new("Error message"),
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
					Id:             new("cmd1"),
					CommandName:    new("test_cmd"),
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
					Id:             new("cmd2"),
					CommandName:    new("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments: []*commanding.CommandAssignment{
						{Name: new("arg1"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("val1")}, UserInput: new(true)},
					},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: new("comment"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("test comment")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with acknowledgements",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             new("cmd3"),
					CommandName:    new("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: new("Acknowledge_Queued_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("OK")}},
						{Name: new("Acknowledge_Queued_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("123")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with extra acknowledgements",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             new("cmd4"),
					CommandName:    new("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: new("Verifier_Test_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("OK")}},
						{Name: new("Verifier_Test_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("456")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command with completion",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             new("cmd5"),
					CommandName:    new("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Assignments:    []*commanding.CommandAssignment{},
					Attr: []*commanding.CommandHistoryAttribute{
						{Name: new("CommandComplete_Status"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("Completed")}},
						{Name: new("CommandComplete_Time"), Value: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("789")}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "Command without id uses command id fallback",
			commands: []*commanding.CommandHistoryEntry{
				{
					CommandName:    new("test_cmd"),
					GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					CommandId: &commanding.CommandId{
						Origin:         new("grafana"),
						SequenceNumber: new(int32(42)),
						GenerationTime: new(int64(123456789)),
						CommandName:    new("test_cmd"),
					},
					Assignments: []*commanding.CommandAssignment{},
					Attr:        []*commanding.CommandHistoryAttribute{},
				},
			},
			wantLen: 1,
		},
		{
			name: "Invalid JSON marshal (but skip)",
			commands: []*commanding.CommandHistoryEntry{
				{
					Id:             new("cmd6"),
					CommandName:    new("test_cmd"),
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
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: new(1.0), Min: new(0.5), Max: new(1.5), N: new(int32(1))},
		}, "param", true, true, 4, 1},
		{"Multiple samples", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: new(1.0), Min: new(0.5), Max: new(1.5), N: new(int32(1))},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), Avg: new(2.0), Min: new(1.5), Max: new(2.5), N: new(int32(1))},
		}, "param", false, false, 2, 2},
		{"With nulls", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), Avg: new(1.0), Min: new(0.5), Max: new(1.5), N: new(int32(1))},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), N: new(int32(0))},
		}, "param", true, true, 4, 2}, // Null appended
		{"Consecutive nulls (but logic appends only one if after non-null)", []*pvalue.TimeSeries_Sample{
			{Time: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), N: new(int32(0))},
			{Time: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), N: new(int32(0))},
			{Time: timestamppb.New(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)), N: new(int32(0))},
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

// TestConvertBufferToFrame tests the ConvertBufferToFrame function.
func TestConvertBufferToFrame(t *testing.T) {
	tests := []struct {
		name       string
		buffer     []*pvalue.ParameterValue
		parameter  string
		includeMin bool
		includeMax bool
		realtime   bool
		wantLen    int
	}{
		{"Empty", []*pvalue.ParameterValue{}, "param", false, false, false, 0},
		{"Empty return default", []*pvalue.ParameterValue{}, "param", false, false, false, 0}, // Checks default frame
		{"With values", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
		}, "param", true, true, false, 1},
		{"Realtime uses now", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
		}, "param", false, false, true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertBufferToFrame(tt.buffer, tt.parameter, tt.includeMin, tt.includeMax, tt.realtime)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
		})
	}
}

// TestConvertRangesToFrame tests the ConvertRangesToFrame function.
func TestConvertRangesToFrame(t *testing.T) {
	tests := []struct {
		name      string
		ranges    *pvalue.Ranges
		parameter string
		wantLen   int
	}{
		{"Nil ranges", nil, "param", 0},
		{"Empty ranges", &pvalue.Ranges{}, "param", 0},
		{"With ranges", &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
			{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_STRING.Enum(), StringValue: new("val1")}}},
		}}, "param", 1},
		{"Empty eng values", &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
			{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{}},
		}}, "param", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertRangesToFrame(tt.ranges, tt.parameter, false)
			assert.Equal(t, tt.wantLen, got.Fields[0].Len())
			if tt.wantLen > 0 {
				assert.Equal(t, "val1", got.Fields[1].At(0))
				assert.Nil(t, got.Fields[0].Config)
				require.NotNil(t, got.Fields[1].Config)
				require.Len(t, got.Fields[1].Config.Mappings, 1)
				mapping := got.Fields[1].Config.Mappings[0].(data.ValueMapper)
				assert.Equal(t, "val1", mapping["val1"].Text)
				assert.Empty(t, mapping["val1"].Color)
			}
		})
	}
}

func TestConvertRangesToFrame_AutomaticColors(t *testing.T) {
	ranges := &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
		{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_ENUMERATED.Enum(), StringValue: new("RUNNING")}}},
		{Start: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: new(true)}}},
	}}

	frame := ConvertRangesToFrame(ranges, "state", true)

	assert.Equal(t, "RUNNING", frame.Fields[1].At(0))
	assert.Equal(t, "true", frame.Fields[1].At(1))
	assert.Nil(t, frame.Fields[0].Config)
	mapping := frame.Fields[1].Config.Mappings[0].(data.ValueMapper)
	assert.Equal(t, "RUNNING", mapping["RUNNING"].Text)
	assert.Regexp(t, "^#[0-9A-F]{6}$", mapping["RUNNING"].Color)
	assert.Equal(t, "TRUE", mapping["true"].Text)
	assert.Equal(t, "#3AAB58", mapping["true"].Color)
}

func TestConvertRangesToFrame_BooleanValues(t *testing.T) {
	ranges := &pvalue.Ranges{Range: []*pvalue.Ranges_Range{
		{Start: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: new(false)}}},
		{Start: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValues: []*protobuf.Value{{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: new(true)}}},
	}}

	frame := ConvertRangesToFrame(ranges, "state", true)

	assert.Equal(t, false, frame.Fields[1].At(0))
	assert.Equal(t, true, frame.Fields[1].At(1))
	assert.Nil(t, frame.Fields[0].Config)
	mapping := frame.Fields[1].Config.Mappings[0].(data.ValueMapper)
	assert.Equal(t, "FALSE", mapping["false"].Text)
	assert.Equal(t, "#D72638", mapping["false"].Color)
	assert.Equal(t, "TRUE", mapping["true"].Text)
	assert.Equal(t, "#3AAB58", mapping["true"].Color)
}

func TestConvertDiscreteBufferToFrame_MatchesRangeMappingShape(t *testing.T) {
	generationTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	buffer := []*pvalue.ParameterValue{
		{
			GenerationTime: timestamppb.New(generationTime),
			EngValue:       &protobuf.Value{Type: protobuf.Value_ENUMERATED.Enum(), StringValue: new("RUNNING")},
		},
	}

	frame := ConvertDiscreteBufferToFrame(buffer, "state", true, false)

	require.Len(t, frame.Fields, 2)
	assert.Equal(t, generationTime, frame.Fields[0].At(0))
	assert.Equal(t, "RUNNING", frame.Fields[1].At(0))
	assert.Nil(t, frame.Fields[0].Config)
	mapping := frame.Fields[1].Config.Mappings[0].(data.ValueMapper)
	assert.Equal(t, "RUNNING", mapping["RUNNING"].Text)
	assert.Regexp(t, "^#[0-9A-F]{6}$", mapping["RUNNING"].Color)
}

// TestConvertBufferToAverageFrame tests the ConvertBufferToAverageFrame function.
func TestConvertBufferToAverageFrame(t *testing.T) {
	tests := []struct {
		name       string
		buffer     []*pvalue.ParameterValue
		parameter  string
		getMin     bool
		getMax     bool
		realtime   bool
		wantFields int
		wantLen    int
	}{
		{"Empty", []*pvalue.ParameterValue{}, "param", false, false, false, 1, 0},
		{"Numeric avg", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(3.0)}},
		}, "param", true, true, false, 4, 1},
		{"String most frequent", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("a")}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("a")}},
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("b")}},
		}, "param", false, false, false, 2, 1},
		{"Realtime", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
		}, "param", false, false, true, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertBufferToAverageFrame(tt.buffer, tt.parameter, tt.getMin, tt.getMax, tt.realtime)
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
		realtime      bool
		wantValuesLen int
		wantTimesLen  int
	}{
		{"Empty", []*pvalue.ParameterValue{}, false, 0, 0},
		{"Basic", []*pvalue.ParameterValue{
			{GenerationTime: timestamppb.New(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)), EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
		}, false, 1, 1},
		{"Realtime", []*pvalue.ParameterValue{
			{EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.0)}},
		}, true, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, times := extractParameterValues(tt.buffer, tt.realtime)
			assert.Equal(t, tt.wantValuesLen, len(values))
			assert.Equal(t, tt.wantTimesLen, len(times))
		})
	}
}

// TestExtractValue tests the extractValue function.
func TestExtractValue(t *testing.T) {
	tests := []struct {
		name string
		v    *protobuf.Value
		want interface{}
	}{
		{"Double", &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(1.5)}, 1.5},
		{"Binary", &protobuf.Value{Type: protobuf.Value_BINARY.Enum(), BinaryValue: []byte{0x01, 0x02}}, "00000001 00000010"},
		{"Timestamp", &protobuf.Value{Type: protobuf.Value_TIMESTAMP.Enum(), TimestampValue: new(int64(123456))}, int64(123456)},
		{"Sint64", &protobuf.Value{Type: protobuf.Value_SINT64.Enum(), Sint64Value: new(int64(-123))}, int64(-123)},
		{"Uint64", &protobuf.Value{Type: protobuf.Value_UINT64.Enum(), Uint64Value: new(uint64(123))}, uint64(123)},
		{"Sint32", &protobuf.Value{Type: protobuf.Value_SINT32.Enum(), Sint32Value: new(int32(-123))}, int32(-123)},
		{"Uint32", &protobuf.Value{Type: protobuf.Value_UINT32.Enum(), Uint32Value: new(uint32(123))}, uint32(123)},
		{"Float", &protobuf.Value{Type: protobuf.Value_FLOAT.Enum(), FloatValue: new(float32(1.5))}, 1.5},
		{"Boolean true", &protobuf.Value{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: new(true)}, "true"},
		{"Boolean false", &protobuf.Value{Type: protobuf.Value_BOOLEAN.Enum(), BooleanValue: new(false)}, "false"},
		{"String", &protobuf.Value{Type: protobuf.Value_STRING.Enum(), StringValue: new("test")}, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractValue(tt.v))
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

// TestCalculateStats tests the calculateStats helper.
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
			avg, min, max := calculateStats(tt.values, tt.parameter)
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
			assert.Equal(t, tt.want, sum(tt.values))
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
			assert.Equal(t, tt.want, sum(tt.values))
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
			min, max := minMax(tt.values)
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
				assert.Equal(t, "", mostFrequent(tt.values))
			} else {
				assert.Equal(t, tt.want, mostFrequent(tt.values))
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

// TestHashToRGB tests the hashToRGB helper.
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
			got := hashToRGB(tt.s)
			assert.Regexp(t, "^#[0-9A-F]{6}$", got)
		})
	}
}

// TestConvertAlarmListToFrame tests the ConvertAlarmListToFrame function.
func TestConvertAlarmListToFrame(t *testing.T) {
	triggerTime := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	updateTime := time.Date(2024, 3, 1, 10, 5, 0, 0, time.UTC)
	ackTime := time.Date(2024, 3, 1, 10, 10, 0, 0, time.UTC)
	shelveTime := time.Date(2024, 3, 1, 10, 15, 0, 0, time.UTC)
	shelveExpiry := time.Date(2024, 3, 1, 11, 15, 0, 0, time.UTC)
	clearTime := time.Date(2024, 3, 1, 10, 20, 0, 0, time.UTC)

	t.Run("Empty alarm list returns frame with zero rows", func(t *testing.T) {
		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{})
		require.NotNil(t, frame)
		assert.Equal(t, "response", frame.Name)
		require.Len(t, frame.Fields, 1)
		assert.Equal(t, "alarms", frame.Fields[0].Name)
		assert.Equal(t, 0, frame.Fields[0].Len())
	})

	t.Run("Parameter alarm - basic fields", func(t *testing.T) {
		alarm := &alarms.AlarmData{
			Type:             alarms.AlarmType_PARAMETER.Enum(),
			Severity:         alarms.AlarmSeverity_WARNING.Enum(),
			TriggerTime:      timestamppb.New(triggerTime),
			UpdateTime:       timestamppb.New(updateTime),
			SeqNum:           new(uint32(42)),
			Violations:       new(uint32(3)),
			Count:            new(uint32(5)),
			Acknowledged:     new(false),
			ProcessOK:        new(false),
			Triggered:        new(true),
			Latching:         new(false),
			NotificationType: alarms.AlarmNotificationType_TRIGGERED.Enum(),
			Id: &protobuf.NamedObjectId{
				Namespace: new("/YSS/SIMULATOR"),
				Name:      new("BatteryVoltage1"),
			},
			ParameterDetail: &alarms.ParameterAlarmData{
				TriggerValue: &pvalue.ParameterValue{
					EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(57.0)},
				},
				CurrentValue: &pvalue.ParameterValue{
					EngValue: &protobuf.Value{Type: protobuf.Value_DOUBLE.Enum(), DoubleValue: new(55.2)},
				},
			},
		}

		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{alarm})
		require.Equal(t, 1, frame.Fields[0].Len())

		var entry AlarmEntry
		err := json.Unmarshal(frame.Fields[0].At(0).(json.RawMessage), &entry)
		require.NoError(t, err)

		assert.Equal(t, "/YSS/SIMULATOR/BatteryVoltage1/42", entry.Id)
		assert.Equal(t, "/YSS/SIMULATOR/BatteryVoltage1", entry.Name)
		assert.Equal(t, "WARNING", entry.Severity)
		assert.Equal(t, "PARAMETER", entry.Type)
		assert.Equal(t, uint32(3), entry.Violations)
		assert.Equal(t, uint32(5), entry.Count)
		assert.Equal(t, uint32(42), entry.SeqNum)
		assert.Equal(t, "Active", entry.State)
		assert.False(t, entry.Acknowledged)
		assert.True(t, entry.Triggered)
		assert.False(t, entry.Latching)
		assert.Equal(t, "TRIGGERED", entry.NotificationType)
		assert.Equal(t, triggerTime.Format(time.RFC3339), entry.TriggerTime)
		assert.Equal(t, updateTime.Format(time.RFC3339), entry.UpdateTime)
		assert.Equal(t, "57.00", entry.TriggerValue)
		assert.Equal(t, "55.20", entry.CurrentValue)
	})

	t.Run("Parameter alarm - with acknowledge info", func(t *testing.T) {
		alarm := &alarms.AlarmData{
			Type:             alarms.AlarmType_PARAMETER.Enum(),
			Severity:         alarms.AlarmSeverity_CRITICAL.Enum(),
			TriggerTime:      timestamppb.New(triggerTime),
			SeqNum:           new(uint32(10)),
			Acknowledged:     new(true),
			ProcessOK:        new(false),
			Triggered:        new(true),
			NotificationType: alarms.AlarmNotificationType_ACKNOWLEDGED.Enum(),
			Id: &protobuf.NamedObjectId{
				Namespace: new("/YSS"),
				Name:      new("Pressure"),
			},
			AcknowledgeInfo: &alarms.AcknowledgeInfo{
				AcknowledgedBy:     new("operator1"),
				AcknowledgeTime:    timestamppb.New(ackTime),
				AcknowledgeMessage: new("Acknowledged, investigating"),
			},
		}

		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{alarm})
		require.Equal(t, 1, frame.Fields[0].Len())

		var entry AlarmEntry
		err := json.Unmarshal(frame.Fields[0].At(0).(json.RawMessage), &entry)
		require.NoError(t, err)

		assert.Equal(t, "CRITICAL", entry.Severity)
		assert.Equal(t, "Acknowledged", entry.State)
		assert.True(t, entry.Acknowledged)
		assert.Equal(t, "operator1", entry.AcknowledgedBy)
		assert.Equal(t, ackTime.Format(time.RFC3339), entry.AcknowledgeTime)
		assert.Equal(t, "Acknowledged, investigating", entry.AcknowledgeComment)
	})

	t.Run("Parameter alarm - shelved fields", func(t *testing.T) {
		alarm := &alarms.AlarmData{
			Type:             alarms.AlarmType_PARAMETER.Enum(),
			Severity:         alarms.AlarmSeverity_WATCH.Enum(),
			TriggerTime:      timestamppb.New(triggerTime),
			SeqNum:           new(uint32(7)),
			Acknowledged:     new(false),
			ProcessOK:        new(false),
			Triggered:        new(true),
			NotificationType: alarms.AlarmNotificationType_SHELVED.Enum(),
			Id: &protobuf.NamedObjectId{
				Namespace: new("/YSS"),
				Name:      new("Temperature"),
			},
			ShelveInfo: &alarms.ShelveInfo{
				ShelvedBy:        new("operator2"),
				ShelveTime:       timestamppb.New(shelveTime),
				ShelveExpiration: timestamppb.New(shelveExpiry),
				ShelveMessage:    new("Known issue, shelved for 1h"),
			},
		}

		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{alarm})
		require.Equal(t, 1, frame.Fields[0].Len())

		var entry AlarmEntry
		err := json.Unmarshal(frame.Fields[0].At(0).(json.RawMessage), &entry)
		require.NoError(t, err)

		assert.Equal(t, "Shelved", entry.State)
		assert.True(t, entry.Shelved)
		assert.Equal(t, "operator2", entry.ShelvedBy)
		assert.Equal(t, shelveTime.Format(time.RFC3339), entry.ShelveTime)
		assert.Equal(t, shelveExpiry.Format(time.RFC3339), entry.ShelveExpiration)
		assert.Equal(t, "Known issue, shelved for 1h", entry.ShelveComment)
	})

	t.Run("Parameter alarm - cleared fields", func(t *testing.T) {
		alarm := &alarms.AlarmData{
			Type:             alarms.AlarmType_PARAMETER.Enum(),
			Severity:         alarms.AlarmSeverity_WARNING.Enum(),
			TriggerTime:      timestamppb.New(triggerTime),
			SeqNum:           new(uint32(99)),
			Acknowledged:     new(true),
			ProcessOK:        new(true),
			Triggered:        new(false),
			NotificationType: alarms.AlarmNotificationType_CLEARED.Enum(),
			Id: &protobuf.NamedObjectId{
				Namespace: new("/YSS"),
				Name:      new("Voltage"),
			},
			ClearInfo: &alarms.ClearInfo{
				ClearedBy:    new("operator3"),
				ClearTime:    timestamppb.New(clearTime),
				ClearMessage: new("Issue resolved"),
			},
		}

		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{alarm})
		require.Equal(t, 1, frame.Fields[0].Len())

		var entry AlarmEntry
		err := json.Unmarshal(frame.Fields[0].At(0).(json.RawMessage), &entry)
		require.NoError(t, err)

		assert.True(t, entry.Cleared)
		assert.Equal(t, "operator3", entry.ClearedBy)
		assert.Equal(t, clearTime.Format(time.RFC3339), entry.ClearTime)
		assert.Equal(t, "Issue resolved", entry.ClearComment)
	})

	t.Run("Event alarm - trigger and current event values", func(t *testing.T) {
		alarm := &alarms.AlarmData{
			Type:             alarms.AlarmType_EVENT.Enum(),
			Severity:         alarms.AlarmSeverity_CRITICAL.Enum(),
			TriggerTime:      timestamppb.New(triggerTime),
			SeqNum:           new(uint32(55)),
			Acknowledged:     new(false),
			ProcessOK:        new(false),
			Triggered:        new(true),
			NotificationType: alarms.AlarmNotificationType_TRIGGERED.Enum(),
			Id: &protobuf.NamedObjectId{
				Namespace: new("/yamcs/event/SystemMonitor"),
				Name:      new("SystemFailure"),
			},
			EventDetail: &alarms.EventAlarmData{
				TriggerEvent: &events.Event{
					Message:  new("Critical system failure detected"),
					Severity: events.Event_CRITICAL.Enum(),
				},
				CurrentEvent: &events.Event{
					Message:  new("System still failing"),
					Severity: events.Event_CRITICAL.Enum(),
				},
			},
		}

		frame := ConvertAlarmListToFrame([]*alarms.AlarmData{alarm})
		require.Equal(t, 1, frame.Fields[0].Len())

		var entry AlarmEntry
		err := json.Unmarshal(frame.Fields[0].At(0).(json.RawMessage), &entry)
		require.NoError(t, err)

		assert.Equal(t, "EVENT", entry.Type)
		assert.Equal(t, "/yamcs/event/SystemMonitor/SystemFailure", entry.Name)
		assert.Equal(t, "CRITICAL: Critical system failure detected", entry.TriggerValue)
		assert.Equal(t, "CRITICAL: System still failing", entry.CurrentValue)
		assert.Equal(t, "Active", entry.State)
	})

	t.Run("Multiple alarms returns correct count", func(t *testing.T) {
		alarmList := []*alarms.AlarmData{
			{
				Type:             alarms.AlarmType_PARAMETER.Enum(),
				Severity:         alarms.AlarmSeverity_WARNING.Enum(),
				TriggerTime:      timestamppb.New(triggerTime),
				SeqNum:           new(uint32(1)),
				NotificationType: alarms.AlarmNotificationType_TRIGGERED.Enum(),
				Id:               &protobuf.NamedObjectId{Namespace: new("/YSS"), Name: new("Param1")},
			},
			{
				Type:             alarms.AlarmType_EVENT.Enum(),
				Severity:         alarms.AlarmSeverity_CRITICAL.Enum(),
				TriggerTime:      timestamppb.New(triggerTime),
				SeqNum:           new(uint32(2)),
				NotificationType: alarms.AlarmNotificationType_TRIGGERED.Enum(),
				Id:               &protobuf.NamedObjectId{Namespace: new("/YSS"), Name: new("Param2")},
			},
		}

		frame := ConvertAlarmListToFrame(alarmList)
		assert.Equal(t, 2, frame.Fields[0].Len())
	})
}
