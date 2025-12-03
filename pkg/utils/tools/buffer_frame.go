package tools

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"golang.org/x/exp/constraints"
)

// ConvertEventsToFrame converts a list of Yamcs events into a Grafana data frame.
func ConvertEventsToFrame(events []*events.Event) *data.Frame {
	messageField := data.NewField("message", nil, []string{})
	severityField := data.NewField("severity", nil, []string{})
	timeField := data.NewField("time", nil, []time.Time{})

	for _, event := range events {
		messageField.Append(event.GetMessage())
		severityField.Append(event.GetSeverity().String())
		timeField.Append(event.GetGenerationTime().AsTime())
	}

	return data.NewFrame("response", timeField, messageField, severityField)
}

type CommandAck struct {
	Status  string `json:"status"`
	Time    string `json:"time"`
	Message string `json:"message,omitempty"`
}

type CommandArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CommandEntry struct {
	Id                    string                 `json:"id"`
	Time                  time.Time              `json:"time"`
	Command               string                 `json:"command"`
	Comment               *string                `json:"comment,omitempty"`
	Arguments             []CommandArgument      `json:"arguments"`
	Queued                *CommandAck            `json:"queued,omitempty"`
	Released              *CommandAck            `json:"released,omitempty"`
	Sent                  *CommandAck            `json:"sent,omitempty"`
	ExtraAcknowledgements map[string]*CommandAck `json:"extraAcks"`
	Completion            *CommandAck            `json:"completion"`
}

// Helpers
func nameHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func nameHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func prepend[T any](s []T, v T) []T {
	return append([]T{v}, s...)
}

func ConvertCommandListToFrame(commands []*commanding.CommandHistoryEntry) *data.Frame {

	commandList := make([]json.RawMessage, 0)

	for _, command := range commands {

		commandEntry := &CommandEntry{
			Id:                    command.GetId(),
			Time:                  command.GetGenerationTime().AsTime(),
			Command:               command.GetCommandName(),
			Comment:               nil,
			Arguments:             make([]CommandArgument, 0),
			Queued:                nil,
			Released:              nil,
			Sent:                  nil,
			Completion:            nil,
			ExtraAcknowledgements: make(map[string]*CommandAck),
		}

		for _, attribute := range command.GetAttr() {
			name := attribute.GetName()
			value := attribute.GetValue()

			switch name {
			case "comment":
				commandEntry.Comment = value.StringValue

			case "Acknowledge_Queued_Status", "Acknowledge_Queued_Time", "Acknowledge_Queued_Message",
				"Acknowledge_Released_Status", "Acknowledge_Released_Time", "Acknowledge_Released_Message",
				"Acknowledge_Sent_Status", "Acknowledge_Sent_Time", "Acknowledge_Sent_Message":

				var ack **CommandAck
				switch {
				case nameHasPrefix(name, "Acknowledge_Queued"):
					ack = &commandEntry.Queued
				case nameHasPrefix(name, "Acknowledge_Released"):
					ack = &commandEntry.Released
				case nameHasPrefix(name, "Acknowledge_Sent"):
					ack = &commandEntry.Sent
				}

				if *ack == nil {
					*ack = &CommandAck{}
				}

				switch {
				case nameHasSuffix(name, "Status"):
					(*ack).Status = value.GetStringValue()
				case nameHasSuffix(name, "Time"):
					(*ack).Time = value.GetStringValue()
				case nameHasSuffix(name, "Message"):
					(*ack).Message = value.GetStringValue()
				}

			default:
				// Handle Verifier_* attributes
				if nameHasPrefix(name, "Verifier_") {
					rest := strings.TrimPrefix(name, "Verifier_")
					underscoreIndex := strings.LastIndex(rest, "_")
					if underscoreIndex > 0 {
						ackName := "Verifier_" + rest[:underscoreIndex]
						field := rest[underscoreIndex+1:]

						ack, ok := commandEntry.ExtraAcknowledgements[ackName]
						if !ok {
							ack = &CommandAck{}
							commandEntry.ExtraAcknowledgements[ackName] = ack
						}

						switch field {
						case "Status":
							ack.Status = value.GetStringValue()
						case "Time":
							ack.Time = value.GetStringValue()
						case "Message":
							ack.Message = value.GetStringValue()
						}
					}
				}

				// Handle CommandComplete_* attributes
				if nameHasPrefix(name, "CommandComplete_") {
					if commandEntry.Completion == nil {
						commandEntry.Completion = &CommandAck{}
					}

					switch {
					case nameHasSuffix(name, "Status"):
						commandEntry.Completion.Status = value.GetStringValue()
					case nameHasSuffix(name, "Time"):
						commandEntry.Completion.Time = value.GetStringValue()
					case nameHasSuffix(name, "Message"):
						commandEntry.Completion.Message = value.GetStringValue()
					}
				}
			}
		}

		for _, assignment := range command.GetAssignments() {
			if !assignment.GetUserInput() {
				continue
			}
			commandEntry.Arguments = append(commandEntry.Arguments, CommandArgument{
				Name:  assignment.GetName(),
				Value: StringifyValue(assignment.GetValue()),
			})
		}

		var rawJson json.RawMessage
		rawJson, err := json.Marshal(commandEntry)
		if err != nil {
			continue
		}
		commandList = prepend(commandList, rawJson)
	}

	return data.NewFrame("response", data.NewField("commands", nil, commandList))
}

// ConvertSampleBufferToFrame converts a time series sample buffer into a data frame.
func ConvertSampleBufferToFrame(buffer []*pvalue.TimeSeries_Sample, parameter string, includeMin, includeMax bool) *data.Frame {

	valueField := data.NewField(parameter, nil, []*float64{})
	minField := data.NewField("min("+parameter+")", nil, []*float64{})
	maxField := data.NewField("max("+parameter+")", nil, []*float64{})
	timeField := data.NewField("time", nil, []time.Time{})

	lastWasNull := false

	for _, item := range buffer {

		if item.GetN() == 0 && !lastWasNull {
			lastWasNull = true
			valueField.Append(nil)
			minField.Append(nil)
			maxField.Append(nil)
			timeField.Append(item.Time.AsTime())
			continue
		} else if item.GetN() == 0 {
			continue
		}
		lastWasNull = false

		timeField.Append(item.Time.AsTime())
		valueField.Append(item.Avg)
		minField.Append(item.Min)
		maxField.Append(item.Max)
	}

	frame := data.NewFrame("response", timeField, valueField)
	if includeMin {
		frame.Fields = append(frame.Fields, minField)
	}
	if includeMax {
		frame.Fields = append(frame.Fields, maxField)
	}
	return frame
}

func ConvertSampleBufferToFrameWithOffset(buffer []*pvalue.TimeSeries_Sample,
	parameter string, includeMin, includeMax bool, offset time.Duration) *data.Frame {

	valueField := data.NewField(parameter, nil, []*float64{})
	minField := data.NewField("min("+parameter+")", nil, []*float64{})
	maxField := data.NewField("max("+parameter+")", nil, []*float64{})
	timeField := data.NewField("time", nil, []time.Time{})

	lastWasNull := false

	for _, item := range buffer {
		timeField.Append(item.Time.AsTime().Add(offset))

		if item.GetN() == 0 && !lastWasNull && valueField.Len() > 0 {
			lastWasNull = true
			valueField.Append(nil)
			minField.Append(nil)
			maxField.Append(nil)
			continue
		}
		lastWasNull = false

		valueField.Append(item.Avg)
		minField.Append(item.Min)
		maxField.Append(item.Max)
	}

	frame := data.NewFrame("response", timeField, valueField)
	if includeMin {
		frame.Fields = append(frame.Fields, minField)
	}
	if includeMax {
		frame.Fields = append(frame.Fields, maxField)
	}
	return frame
}

// ConvertBufferToFrame converts a parameter value buffer into a data frame.
func ConvertBufferToFrame(buffer []*pvalue.ParameterValue, parameter string, includeMin, includeMax bool, aggregatePath string, realtime bool) *data.Frame {
	if len(buffer) == 0 {
		return data.NewFrame("response", data.NewField("time", nil, []time.Time{}), data.NewField(parameter, nil, []int32{}))
	}

	values, times := extractParameterValues(buffer, aggregatePath, realtime)
	valueField := CreateValueField(values, parameter)

	frame := data.NewFrame("response", data.NewField("time", nil, times), valueField)
	if includeMin || includeMax {
		_, minField, maxField := CalculateStats(values, parameter)
		if includeMin {
			frame.Fields = append(frame.Fields, minField)
		}
		if includeMax {
			frame.Fields = append(frame.Fields, maxField)
		}
	}
	return frame
}

// ConvertRangesToFrame converts a range of parameter values into a Grafana data frame.
func ConvertRangesToFrame(ranges *pvalue.Ranges, parameter string, aggregatePath string) *data.Frame {

	times := []time.Time{}
	values := []interface{}{}
	labels := data.Labels{}
	valueMapping := data.ValueMapper{}
	valueMapping["true"] = data.ValueMappingResult{
		Text:  "TRUE",
		Color: "#3AAB58",
	}
	valueMapping["false"] = data.ValueMappingResult{
		Text:  "FALSE",
		Color: "#D72638",
	}

	for _, valueRange := range ranges.GetRange() {
		if len(valueRange.GetEngValues()) > 0 {
			val := extractValue(valueRange.GetEngValues()[0], aggregatePath)
			label := fmt.Sprint(val)
			labels[label] = label
			valueMapping[label] = data.ValueMappingResult{
				Color: HashToRGB(label),
			}
			values = append(values, val)
			times = append(times, valueRange.GetStart().AsTime())
		}
	}

	valueField := CreateValueField(values, parameter)
	valueField.Config = &data.FieldConfig{}
	valueField.Config.Mappings = []data.ValueMapping{valueMapping}
	timeField := data.NewField("time", labels, times)
	return data.NewFrame("response", timeField, valueField)
}

// ConvertBufferToAverageFrame extracts statistics from the parameter buffer and returns a data frame.
func ConvertBufferToAverageFrame(buffer []*pvalue.ParameterValue,
	parameter string, getMin, getMax bool, aggregatePath string, realtime bool) *data.Frame {
	if len(buffer) == 0 {
		return data.NewFrame("response", data.NewField("time", nil, []time.Time{}))
	}

	values, times := extractParameterValues(buffer, aggregatePath, realtime)
	avg, min, max := CalculateStats(values, parameter)

	timeField := data.NewField("time", nil, []time.Time{times[len(times)-1]})
	if realtime {
		timeField = data.NewField("time", nil, []time.Time{time.Now()})
	}
	frame := data.NewFrame("response", timeField, avg)

	if getMin {
		frame.Fields = append(frame.Fields, min)
	}
	if getMax {
		frame.Fields = append(frame.Fields, max)
	}

	return frame
}

// extractParameterValues extracts values and timestamps from a parameter buffer.
func extractParameterValues(buffer []*pvalue.ParameterValue, aggregatePath string, realtime bool) ([]interface{}, []time.Time) {
	var values []interface{}
	var times []time.Time

	for _, item := range buffer {
		values = append(values, extractValue(item.GetEngValue(), aggregatePath))
		if realtime {
			times = append(times, time.Now())
		} else {
			times = append(times, item.GetGenerationTime().AsTime())
		}

	}
	return values, times
}

// extractValue extracts the correct value type from a Yamcs parameter value.
func extractValue(v *protobuf.Value, aggregatePath string) interface{} {
	switch v.GetType() {
	case protobuf.Value_DOUBLE:
		return v.GetDoubleValue()
	case protobuf.Value_BINARY:
		return formatBinary(v.GetBinaryValue())
	case protobuf.Value_TIMESTAMP:
		return v.GetTimestampValue()
	case protobuf.Value_SINT64:
		return v.GetSint64Value()
	case protobuf.Value_UINT64:
		return v.GetUint64Value()
	case protobuf.Value_SINT32:
		return v.GetSint32Value()
	case protobuf.Value_UINT32:
		return v.GetUint32Value()
	case protobuf.Value_FLOAT:
		return float64(v.GetFloatValue())
	case protobuf.Value_BOOLEAN:
		return strconv.FormatBool(v.GetBooleanValue())
	case protobuf.Value_AGGREGATE:
		return extractValue(aggregateExtractFromPath(v, aggregatePath), "")
	default:
		return v.GetStringValue()
	}
}

// splitPath splits a path like ".x[0].y[2]" into its components.
func splitPath(path string) []string {
	re := regexp.MustCompile(`\.?([a-zA-Z_]\w*|\[\d+\])`)
	matches := re.FindAllString(path, -1)
	for i, match := range matches {
		matches[i] = strings.TrimPrefix(match, ".")
	}
	return matches
}

func aggregateExtractFromPath(value *protobuf.Value, path string) *protobuf.Value {

	defaultValue := &protobuf.Value{Type: protobuf.Value_SINT64.Enum(), Sint64Value: new(int64)}

	if path == "" {
		return defaultValue
	}

	paths := splitPath(path)
	for _, p := range paths {
		if value.GetType() == protobuf.Value_ARRAY && strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			indexStr := p[1 : len(p)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return defaultValue
			}
			arrayValue := value.GetArrayValue()
			if index < len(arrayValue) {
				value = arrayValue[index]
			} else {
				return defaultValue
			}
		} else if value.GetType() == protobuf.Value_AGGREGATE {
			found := false
			aggregateValue := value.GetAggregateValue()
			for i, name := range aggregateValue.GetName() {
				if strings.EqualFold(name, p) {
					value = aggregateValue.GetValue()[i]
					found = true
					break
				}
			}
			if !found {
				return defaultValue
			}
		}
	}
	return value
}

// formatBinary converts a binary value into a readable string.
func formatBinary(data []byte) string {
	var binaryStr string
	for _, b := range data {
		binaryStr += fmt.Sprintf("%08b ", b)
	}
	if len(binaryStr) == 0 {
		return binaryStr
	}
	return binaryStr[:len(binaryStr)-1]
}

// CreateValueField generates a Grafana field for the given values.
func CreateValueField(values []interface{}, parameter string) *data.Field {
	if len(values) == 0 {
		return data.NewField(parameter, nil, []string{})
	}

	switch values[0].(type) {
	case int64:
		return data.NewField(parameter, nil, ConvertSlice[int64](values))
	case uint64:
		return data.NewField(parameter, nil, ConvertSlice[uint64](values))
	case int32:
		return data.NewField(parameter, nil, ConvertSlice[int32](values))
	case uint32:
		return data.NewField(parameter, nil, ConvertSlice[uint32](values))
	case float64:
		return data.NewField(parameter, nil, ConvertSlice[float64](values))
	case bool:
		return data.NewField(parameter, nil, ConvertSlice[bool](values))
	default:
		return data.NewField(parameter, nil, ConvertSlice[string](values))
	}
}

// ConvertSlice is a generic function to convert []interface{} to []T.
func ConvertSlice[T any](values []interface{}) []T {
	result := make([]T, len(values))
	for i, v := range values {
		result[i] = v.(T)
	}
	return result
}

func convert[T any](values []interface{}) []T {
	result := make([]T, len(values))
	for i, v := range values {
		result[i] = v.(T)
	}
	return result
}

// CalculateStats computes the average, min, and max values based on type.
func CalculateStats(values []interface{}, parameter string) (*data.Field, *data.Field, *data.Field) {

	if len(values) == 0 {
		return data.NewField(parameter, nil, []float64{}),
			data.NewField("min("+parameter+")", nil, []float64{}),
			data.NewField("max("+parameter+")", nil, []float64{})
	}

	switch values[0].(type) {
	case int64:
		vals := convert[int64](values)
		min, max := MinMax(vals)
		return createStatFields(parameter, vals, Sum(vals), min, max)
	case uint64:
		vals := convert[uint64](values)
		min, max := MinMax(vals)
		return createStatFields(parameter, vals, Sum(vals), min, max)
	case int32:
		vals := convert[int32](values)
		min, max := MinMax(vals)
		return createStatFields(parameter, vals, Sum(vals), min, max)
	case uint32:
		vals := convert[uint32](values)
		min, max := MinMax(vals)
		return createStatFields(parameter, vals, Sum(vals), min, max)
	case float64:
		vals := convert[float64](values)
		min, max := MinMax(vals)
		return createStatFields(parameter, vals, Sum(vals), min, max)
	case string:
		mostFrequent := MostFrequent(values).(string)
		labels := data.Labels{}
		labels[mostFrequent] = mostFrequent
		valueField := data.NewField(parameter, labels, []string{mostFrequent})
		valueField.Config = &data.FieldConfig{}
		valueMapping := data.ValueMapper{}
		if (mostFrequent == "true") || (mostFrequent == "false") {
			valueMapping["true"] = data.ValueMappingResult{
				Text:  "TRUE",
				Color: "#3AAB58",
			}
			valueMapping["false"] = data.ValueMappingResult{
				Text:  "FALSE",
				Color: "#D72638",
			}
		}
		valueMapping[mostFrequent] = data.ValueMappingResult{
			Color: HashToRGB(mostFrequent),
		}
		valueField.Config.Mappings = []data.ValueMapping{valueMapping}
		return valueField, nil, nil
	default:
		return data.NewField(parameter, nil, []float64{}),
			data.NewField("min("+parameter+")", nil, []float64{}),
			data.NewField("max("+parameter+")", nil, []float64{})
	}
}

func createStatFields[T constraints.Float | constraints.Integer](param string, values []T, sum T, min T, max T) (*data.Field, *data.Field, *data.Field) {
	avg := float64(sum) / float64(len(values))
	return data.NewField(param, nil, []float64{avg}),
		data.NewField("min("+param+")", nil, []T{min}),
		data.NewField("max("+param+")", nil, []T{max})
}

func Sum[T constraints.Float | constraints.Integer](values []T) T {
	var sum T
	for _, v := range values {
		sum += v
	}
	return sum
}

func MinMax[T constraints.Float | constraints.Integer](values []T) (T, T) {
	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// MostFrequent finds the most frequent value in a slice.
func MostFrequent[T comparable](values []T) T {
	freq := make(map[T]int)
	var maxCount int
	var mostFrequent T

	for _, v := range values {
		freq[v]++
		if freq[v] > maxCount {
			maxCount = freq[v]
			mostFrequent = v
		}
	}
	return mostFrequent
}

// hashString generates a numeric hash from a string
func hashString(s string) int {
	hash := md5.Sum([]byte(s)) // Use MD5 for a stable hash
	return int(hash[0])<<24 | int(hash[1])<<16 | int(hash[2])<<8 | int(hash[3])
}

// hslToRgb converts HSL (hue, saturation, lightness) to RGB
func hslToRgb(h, s, l float64) (int, int, int) {
	s /= 100
	l /= 100

	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return int((r + m) * 255), int((g + m) * 255), int((b + m) * 255)
}

// hashToRGB generates a deterministic RGB color from a string
func HashToRGB(name string) string {
	hash := hashString(name)
	hue := float64(hash % 360) // Generate a hue between 0-360
	r, g, b := hslToRgb(hue, 70, 50)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b) // Format as hex string
}
