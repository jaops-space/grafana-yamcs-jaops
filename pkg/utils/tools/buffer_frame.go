package tools

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"golang.org/x/exp/constraints"
	"google.golang.org/protobuf/encoding/protojson"
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

// AlarmEntry represents a processed alarm for the frontend
type AlarmEntry struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	TriggerTime        string `json:"triggerTime"`
	UpdateTime         string `json:"updateTime,omitempty"`
	Severity           string `json:"severity"`
	Type               string `json:"type"`
	Violations         uint32 `json:"violations"`
	Count              uint32 `json:"count"`
	State              string `json:"state"`
	Acknowledged       bool   `json:"acknowledged"`
	AcknowledgedBy     string `json:"acknowledgedBy,omitempty"`
	AcknowledgeTime    string `json:"acknowledgeTime,omitempty"`
	AcknowledgeComment string `json:"acknowledgeComment,omitempty"`
	ProcessOK          bool   `json:"processOK"`
	Triggered          bool   `json:"triggered"`
	Latching           bool   `json:"latching"`
	Shelved            bool   `json:"shelved"`
	ShelvedBy          string `json:"shelvedBy,omitempty"`
	ShelveTime         string `json:"shelveTime,omitempty"`
	ShelveExpiration   string `json:"shelveExpiration,omitempty"`
	ShelveComment      string `json:"shelveComment,omitempty"`
	Cleared            bool   `json:"cleared,omitempty"`
	ClearedBy          string `json:"clearedBy,omitempty"`
	ClearTime          string `json:"clearTime,omitempty"`
	ClearComment       string `json:"clearComment,omitempty"`
	CurrentValue       string `json:"currentValue,omitempty"`
	TriggerValue       string `json:"triggerValue,omitempty"`
	MostSevereValue    string `json:"mostSevereValue,omitempty"`
	// Detailed parameter value objects for inspection
	TriggerValueDetail    interface{} `json:"triggerValueDetail,omitempty"`
	MostSevereValueDetail interface{} `json:"mostSevereValueDetail,omitempty"`
	CurrentValueDetail    interface{} `json:"currentValueDetail,omitempty"`
	ParameterInfo         interface{} `json:"parameterInfo,omitempty"`
	NotificationType      string      `json:"notificationType"`
	SeqNum                uint32      `json:"seqNum"`
}

// ConvertAlarmListToFrame converts a list of Yamcs alarms into a Grafana data frame.
func ConvertAlarmListToFrame(alarmList []*alarms.AlarmData) *data.Frame {
	alarmEntries := make([]json.RawMessage, 0)

	for _, alarm := range alarmList {
		// Construct the full qualified name from namespace and short name.
		// Yamcs returns id.name as short name (e.g. "BatteryVoltage1") and
		// id.namespace as the path (e.g. "/YSS/SIMULATOR"). The Edit Alarm
		// API requires the full qualified name in the URL path.
		alarmId := alarm.GetId()
		if alarmId == nil {
			continue
		}
		qualifiedName := alarmId.GetNamespace() + "/" + alarmId.GetName()

		// Guard TriggerTime: use zero time as fallback if nil
		triggerTime := time.Time{}
		if tt := alarm.GetTriggerTime(); tt != nil {
			triggerTime = tt.AsTime()
		}

		alarmEntry := &AlarmEntry{
			Id:               fmt.Sprintf("%s/%d", qualifiedName, alarm.GetSeqNum()),
			Name:             qualifiedName,
			TriggerTime:      triggerTime.Format(time.RFC3339),
			Severity:         alarm.GetSeverity().String(),
			Type:             alarm.GetType().String(),
			Violations:       alarm.GetViolations(),
			Count:            alarm.GetCount(),
			State:            deriveAlarmState(alarm),
			Acknowledged:     alarm.GetAcknowledged(),
			ProcessOK:        alarm.GetProcessOK(),
			Triggered:        alarm.GetTriggered(),
			Latching:         alarm.GetLatching(),
			NotificationType: alarm.GetNotificationType().String(),
			SeqNum:           alarm.GetSeqNum(),
			Shelved:          alarm.GetShelveInfo() != nil,
		}

		if alarm.GetUpdateTime() != nil {
			alarmEntry.UpdateTime = alarm.GetUpdateTime().AsTime().Format(time.RFC3339)
		}

		if ackInfo := alarm.GetAcknowledgeInfo(); ackInfo != nil {
			alarmEntry.AcknowledgedBy = ackInfo.GetAcknowledgedBy()
			if ackInfo.GetAcknowledgeTime() != nil {
				alarmEntry.AcknowledgeTime = ackInfo.GetAcknowledgeTime().AsTime().Format(time.RFC3339)
			}
			alarmEntry.AcknowledgeComment = ackInfo.GetAcknowledgeMessage()
		}

		// Extract values for parameter alarms
		if paramDetail := alarm.GetParameterDetail(); paramDetail != nil {
			// Extract stringified values for display
			if currentVal := paramDetail.GetCurrentValue(); currentVal != nil {
				alarmEntry.CurrentValue = StringifyValue(currentVal.GetEngValue())
				alarmEntry.CurrentValueDetail = convertParameterValueToMap(currentVal)
			}
			if triggerVal := paramDetail.GetTriggerValue(); triggerVal != nil {
				alarmEntry.TriggerValue = StringifyValue(triggerVal.GetEngValue())
				alarmEntry.TriggerValueDetail = convertParameterValueToMap(triggerVal)
			}
			if mostSevereVal := paramDetail.GetMostSevereValue(); mostSevereVal != nil {
				alarmEntry.MostSevereValue = StringifyValue(mostSevereVal.GetEngValue())
				alarmEntry.MostSevereValueDetail = convertParameterValueToMap(mostSevereVal)
			}
			// Extract parameter info
			if paramInfo := paramDetail.GetParameter(); paramInfo != nil {
				alarmEntry.ParameterInfo = convertParameterInfoToMap(paramInfo)
			}
		}

		// Extract values for event alarms
		if eventDetail := alarm.GetEventDetail(); eventDetail != nil {
			// For event alarms, use the event message and severity as the "values"
			if triggerEvent := eventDetail.GetTriggerEvent(); triggerEvent != nil {
				alarmEntry.TriggerValue = fmt.Sprintf("%s: %s", triggerEvent.GetSeverity().String(), triggerEvent.GetMessage())
			}
			if currentEvent := eventDetail.GetCurrentEvent(); currentEvent != nil {
				alarmEntry.CurrentValue = fmt.Sprintf("%s: %s", currentEvent.GetSeverity().String(), currentEvent.GetMessage())
			}
		}

		// Shelve info
		if shelveInfo := alarm.GetShelveInfo(); shelveInfo != nil {
			alarmEntry.ShelvedBy = shelveInfo.GetShelvedBy()
			if shelveInfo.GetShelveTime() != nil {
				alarmEntry.ShelveTime = shelveInfo.GetShelveTime().AsTime().Format(time.RFC3339)
			}
			if shelveInfo.GetShelveExpiration() != nil {
				alarmEntry.ShelveExpiration = shelveInfo.GetShelveExpiration().AsTime().Format(time.RFC3339)
			}
			alarmEntry.ShelveComment = shelveInfo.GetShelveMessage()
		}

		// Clear info
		if clearInfo := alarm.GetClearInfo(); clearInfo != nil {
			alarmEntry.Cleared = true
			alarmEntry.ClearedBy = clearInfo.GetClearedBy()
			if clearInfo.GetClearTime() != nil {
				alarmEntry.ClearTime = clearInfo.GetClearTime().AsTime().Format(time.RFC3339)
			}
			alarmEntry.ClearComment = clearInfo.GetClearMessage()
		}

		rawJson, err := json.Marshal(alarmEntry)
		if err != nil {
			continue
		}
		alarmEntries = append(alarmEntries, rawJson)
	}

	return data.NewFrame("response", data.NewField("alarms", nil, alarmEntries))
}

// deriveAlarmState returns a string state for the alarm, matching Yamcs web vocabulary.
func deriveAlarmState(alarm *alarms.AlarmData) string {
	if alarm.GetClearInfo() != nil {
		return "Cleared"
	}
	if alarm.GetShelveInfo() != nil {
		return "Shelved"
	}
	if alarm.GetAcknowledged() {
		return "Acknowledged"
	}
	if alarm.GetSeverity().String() != "" {
		return "Active"
	}
	return "Unknown"
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

func commandHistoryEntryID(command *commanding.CommandHistoryEntry) string {
	if command.GetId() != "" {
		return command.GetId()
	}

	commandID := command.GetCommandId()
	if commandID == nil {
		return ""
	}

	return fmt.Sprintf(
		"%s/%d/%d/%s",
		commandID.GetOrigin(),
		commandID.GetSequenceNumber(),
		commandID.GetGenerationTime(),
		commandID.GetCommandName(),
	)
}

func ConvertCommandListToFrame(commands []*commanding.CommandHistoryEntry) *data.Frame {

	commandList := make([]json.RawMessage, 0)

	for _, command := range commands {
		backend.Logger.Info("received command history entry", "entry", protojson.Format(command))

		commandEntry := &CommandEntry{
			Id:                    commandHistoryEntryID(command),
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
				case strings.HasPrefix(name, "Acknowledge_Queued"):
					backend.Logger.Debug("received queued!")
					ack = &commandEntry.Queued
				case strings.HasPrefix(name, "Acknowledge_Released"):
					backend.Logger.Debug("received released!")
					ack = &commandEntry.Released
				case strings.HasPrefix(name, "Acknowledge_Sent"):
					backend.Logger.Debug("received sent!")
					ack = &commandEntry.Sent
				}

				if *ack == nil {
					*ack = &CommandAck{}
				}

				switch {
				case strings.HasSuffix(name, "Status"):
					(*ack).Status = value.GetStringValue()
				case strings.HasSuffix(name, "Time"):
					(*ack).Time = value.GetStringValue()
				case strings.HasSuffix(name, "Message"):
					(*ack).Message = value.GetStringValue()
				}

			default:
				// Handle Verifier_* attributes
				if strings.HasPrefix(name, "Verifier_") {
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
				if strings.HasPrefix(name, "CommandComplete_") {
					if commandEntry.Completion == nil {
						commandEntry.Completion = &CommandAck{}
					}

					switch {
					case strings.HasSuffix(name, "Status"):
						commandEntry.Completion.Status = value.GetStringValue()
					case strings.HasSuffix(name, "Time"):
						commandEntry.Completion.Time = value.GetStringValue()
					case strings.HasSuffix(name, "Message"):
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
		commandList = append(commandList, rawJson)
	}

	for i, j := 0, len(commandList)-1; i < j; i, j = i+1, j-1 {
		commandList[i], commandList[j] = commandList[j], commandList[i]
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

// ConvertBufferToFrame converts a parameter value buffer into a data frame.
func ConvertBufferToFrame(buffer []*pvalue.ParameterValue, parameter string, includeMin, includeMax bool, realtime bool) *data.Frame {
	if len(buffer) == 0 {
		return data.NewFrame("response", data.NewField("time", nil, []time.Time{}), data.NewField(parameter, nil, []int32{}))
	}

	values, times := extractParameterValues(buffer, realtime)
	valueField := CreateValueField(values, parameter)

	frame := data.NewFrame("response", data.NewField("time", nil, times), valueField)
	if includeMin || includeMax {
		_, minField, maxField := calculateStats(values, parameter)
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
func ConvertRangesToFrame(ranges *pvalue.Ranges, parameter string, automaticColors bool) *data.Frame {
	times := []time.Time{}
	values := []interface{}{}
	valueMapping := data.ValueMapper{}

	for _, valueRange := range ranges.GetRange() {
		if len(valueRange.GetEngValues()) == 0 {
			continue
		}

		value := valueRange.GetEngValues()[0]
		frameValue, label := discreteFrameValueAndLabel(value)
		values = append(values, frameValue)
		times = append(times, valueRange.GetStart().AsTime())

		mappingKey := valueMappingKey(frameValue)
		if _, exists := valueMapping[mappingKey]; !exists {
			valueMapping[mappingKey] = discreteValueMapping(label, value.GetType(), automaticColors)
		}
	}

	valueField := CreateValueField(values, parameter)
	valueField.Config = &data.FieldConfig{Mappings: []data.ValueMapping{valueMapping}}
	timeField := data.NewField("time", nil, times)
	return data.NewFrame("response", timeField, valueField)
}

// ConvertDiscreteBufferToFrame converts realtime parameter values to the same
// discrete field/value-mapping shape used by historical range frames.
func ConvertDiscreteBufferToFrame(buffer []*pvalue.ParameterValue, parameter string, automaticColors bool, realtime bool) *data.Frame {
	times := []time.Time{}
	values := []interface{}{}
	valueMapping := data.ValueMapper{}

	for _, item := range buffer {
		value := item.GetEngValue()
		frameValue, label := discreteFrameValueAndLabel(value)
		values = append(values, frameValue)
		if realtime {
			times = append(times, time.Now())
		} else {
			times = append(times, item.GetGenerationTime().AsTime())
		}

		mappingKey := valueMappingKey(frameValue)
		if _, exists := valueMapping[mappingKey]; !exists {
			valueMapping[mappingKey] = discreteValueMapping(label, value.GetType(), automaticColors)
		}
	}

	valueField := CreateValueField(values, parameter)
	valueField.Config = &data.FieldConfig{Mappings: []data.ValueMapping{valueMapping}}
	timeField := data.NewField("time", nil, times)
	return data.NewFrame("response", timeField, valueField)
}

// ConvertBufferToAverageFrame extracts statistics from the parameter buffer and returns a data frame.
func ConvertBufferToAverageFrame(buffer []*pvalue.ParameterValue,
	parameter string, getMin, getMax bool, realtime bool) *data.Frame {
	if len(buffer) == 0 {
		return data.NewFrame("response", data.NewField("time", nil, []time.Time{}))
	}

	values, times := extractParameterValues(buffer, realtime)
	avg, min, max := calculateStats(values, parameter)

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
func extractParameterValues(buffer []*pvalue.ParameterValue, realtime bool) ([]interface{}, []time.Time) {
	var values []interface{}
	var times []time.Time

	for _, item := range buffer {
		values = append(values, extractValue(item.GetEngValue()))
		if realtime {
			times = append(times, time.Now())
		} else {
			times = append(times, item.GetGenerationTime().AsTime())
		}

	}
	return values, times
}

// extractValue extracts the correct value type from a Yamcs parameter value.
func extractValue(v *protobuf.Value) interface{} {
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
		return v.String()
	default:
		return v.GetStringValue()
	}
}

func discreteFrameValueAndLabel(v *protobuf.Value) (interface{}, string) {
	if v == nil {
		return "", ""
	}

	switch v.GetType() {
	case protobuf.Value_STRING, protobuf.Value_ENUMERATED:
		value := v.GetStringValue()
		return value, value
	case protobuf.Value_BOOLEAN:
		value := v.GetBooleanValue()
		label := strconv.FormatBool(value)
		return value, label
	case protobuf.Value_UINT32:
		value := v.GetUint32Value()
		return value, strconv.FormatUint(uint64(value), 10)
	case protobuf.Value_SINT32:
		value := v.GetSint32Value()
		return value, strconv.FormatInt(int64(value), 10)
	case protobuf.Value_UINT64:
		value := v.GetUint64Value()
		return value, strconv.FormatUint(value, 10)
	case protobuf.Value_SINT64:
		value := v.GetSint64Value()
		return value, strconv.FormatInt(value, 10)
	default:
		value := extractValue(v)
		return value, fmt.Sprint(value)
	}
}

func valueMappingKey(value interface{}) string {
	return fmt.Sprint(value)
}

func discreteValueMapping(label string, valueType protobuf.Value_Type, automaticColors bool) data.ValueMappingResult {
	result := data.ValueMappingResult{Text: label}
	if !automaticColors {
		return result
	}

	if valueType == protobuf.Value_BOOLEAN {
		switch strings.ToLower(label) {
		case "true":
			result.Color = "#3AAB58"
		case "false":
			result.Color = "#D72638"
		}
		return result
	}

	result.Color = hashToRGB(label)
	return result
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
		return data.NewField(parameter, nil, convertSliceOrString[int64](values))
	case uint64:
		return data.NewField(parameter, nil, convertSliceOrString[uint64](values))
	case int32:
		return data.NewField(parameter, nil, convertSliceOrString[int32](values))
	case uint32:
		return data.NewField(parameter, nil, convertSliceOrString[uint32](values))
	case float64:
		return data.NewField(parameter, nil, convertSliceOrString[float64](values))
	case bool:
		return data.NewField(parameter, nil, convertSliceOrString[bool](values))
	default:
		return data.NewField(parameter, nil, stringifySlice(values))
	}
}

// convertSlice converts []interface{} to []T.
func convertSlice[T any](values []interface{}) []T {
	result := make([]T, len(values))
	for i, v := range values {
		result[i] = v.(T)
	}
	return result
}

func convertSliceOrString[T any](values []interface{}) interface{} {
	result := make([]T, len(values))
	for i, v := range values {
		typed, ok := v.(T)
		if !ok {
			return stringifySlice(values)
		}
		result[i] = typed
	}
	return result
}

func stringifySlice(values []interface{}) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = fmt.Sprint(v)
	}
	return result
}

// calculateStats computes the average, min, and max values based on type.
func calculateStats(values []interface{}, parameter string) (*data.Field, *data.Field, *data.Field) {

	if len(values) == 0 {
		return data.NewField(parameter, nil, []float64{}),
			data.NewField("min("+parameter+")", nil, []float64{}),
			data.NewField("max("+parameter+")", nil, []float64{})
	}

	switch values[0].(type) {
	case int64:
		vals := convertSlice[int64](values)
		min, max := minMax(vals)
		return createStatFields(parameter, vals, sum(vals), min, max)
	case uint64:
		vals := convertSlice[uint64](values)
		min, max := minMax(vals)
		return createStatFields(parameter, vals, sum(vals), min, max)
	case int32:
		vals := convertSlice[int32](values)
		min, max := minMax(vals)
		return createStatFields(parameter, vals, sum(vals), min, max)
	case uint32:
		vals := convertSlice[uint32](values)
		min, max := minMax(vals)
		return createStatFields(parameter, vals, sum(vals), min, max)
	case float64:
		vals := convertSlice[float64](values)
		min, max := minMax(vals)
		return createStatFields(parameter, vals, sum(vals), min, max)
	case string:
		mostFrequent := mostFrequent(values).(string)
		labels := data.Labels{}
		labels[mostFrequent] = mostFrequent
		valueField := data.NewField(parameter, labels, []string{mostFrequent})
		valueField.Config = &data.FieldConfig{}
		valueField.Config.Mappings = []data.ValueMapping{coloredValueMapping(mostFrequent)}
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

func sum[T constraints.Float | constraints.Integer](values []T) T {
	var sum T
	for _, v := range values {
		sum += v
	}
	return sum
}

func minMax[T constraints.Float | constraints.Integer](values []T) (T, T) {
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

// mostFrequent finds the most frequent value in a slice.
func mostFrequent[T comparable](values []T) T {
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
func hashToRGB(name string) string {
	hash := hashString(name)
	hue := float64(hash % 360) // Generate a hue between 0-360
	r, g, b := hslToRgb(hue, 70, 50)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b) // Format as hex string
}

func coloredValueMapping(label string) data.ValueMapper {
	valueMapping := data.ValueMapper{
		label: {Color: hashToRGB(label)},
	}
	if label == "true" || label == "false" {
		valueMapping["true"] = data.ValueMappingResult{Text: "true", Color: "#3AAB58"}
		valueMapping["false"] = data.ValueMappingResult{Text: "false", Color: "#D72638"}
	}
	return valueMapping
}

// convertParameterValueToMap converts a ParameterValue protobuf to a JSON-serializable map
func convertParameterValueToMap(pv *pvalue.ParameterValue) map[string]interface{} {
	if pv == nil {
		return nil
	}

	result := make(map[string]interface{})

	if id := pv.GetId(); id != nil {
		result["id"] = map[string]interface{}{
			"namespace": id.GetNamespace(),
			"name":      id.GetName(),
		}
	}

	if engValue := pv.GetEngValue(); engValue != nil {
		result["engValue"] = convertValueToInterface(engValue)
	}

	if rawValue := pv.GetRawValue(); rawValue != nil {
		result["rawValue"] = convertValueToInterface(rawValue)
	}

	if acqTime := pv.GetAcquisitionTime(); acqTime != nil {
		result["acquisitionTime"] = acqTime.AsTime().Format(time.RFC3339)
	}

	if genTime := pv.GetGenerationTime(); genTime != nil {
		result["generationTime"] = genTime.AsTime().Format(time.RFC3339)
	}

	if acqStatus := pv.GetAcquisitionStatus(); acqStatus != 0 {
		result["acquisitionStatus"] = acqStatus.String()
	}

	if monResult := pv.GetMonitoringResult(); monResult != 0 {
		result["monitoringResult"] = monResult.String()
	}

	if rangeCondition := pv.GetRangeCondition(); rangeCondition != 0 {
		result["rangeCondition"] = rangeCondition.String()
	}

	if expireMs := pv.GetExpireMillis(); expireMs != 0 {
		result["expireMillis"] = expireMs
	}

	return result
}

// convertValueToInterface converts a protobuf Value to a basic Go type
func convertValueToInterface(v *protobuf.Value) interface{} {
	if v == nil {
		return nil
	}

	switch v.GetType() {
	case protobuf.Value_DOUBLE:
		return v.GetDoubleValue()
	case protobuf.Value_FLOAT:
		return v.GetFloatValue()
	case protobuf.Value_SINT32:
		return v.GetSint32Value()
	case protobuf.Value_UINT32:
		return v.GetUint32Value()
	case protobuf.Value_SINT64:
		return v.GetSint64Value()
	case protobuf.Value_UINT64:
		return v.GetUint64Value()
	case protobuf.Value_BOOLEAN:
		return v.GetBooleanValue()
	case protobuf.Value_STRING:
		return v.GetStringValue()
	case protobuf.Value_BINARY:
		return formatBinary(v.GetBinaryValue())
	case protobuf.Value_TIMESTAMP:
		return v.GetTimestampValue()
	case protobuf.Value_AGGREGATE:
		agg := v.GetAggregateValue()
		result := make(map[string]interface{})
		for i, name := range agg.GetName() {
			if i < len(agg.GetValue()) {
				result[name] = convertValueToInterface(agg.GetValue()[i])
			}
		}
		return result
	case protobuf.Value_ARRAY:
		arr := v.GetArrayValue()
		result := make([]interface{}, len(arr))
		for i, val := range arr {
			result[i] = convertValueToInterface(val)
		}
		return result
	default:
		return v.GetStringValue()
	}
}

// convertParameterInfoToMap converts a ParameterInfo protobuf to a JSON-serializable map
func convertParameterInfoToMap(pi interface{}) map[string]interface{} {
	if pi == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Try to marshal and unmarshal to get a JSON representation
	jsonBytes, err := json.Marshal(pi)
	if err != nil {
		return result
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		return result
	}

	// Extract only the most relevant fields to avoid clutter
	if qualifiedName, ok := jsonMap["qualifiedName"]; ok {
		result["qualifiedName"] = qualifiedName
	}
	if dataSource, ok := jsonMap["dataSource"]; ok {
		result["dataSource"] = dataSource
	}
	if typeInfo, ok := jsonMap["type"]; ok {
		result["type"] = typeInfo
	}
	if shortDesc, ok := jsonMap["shortDescription"]; ok {
		result["shortDescription"] = shortDesc
	}
	if longDesc, ok := jsonMap["longDescription"]; ok {
		result["longDescription"] = longDesc
	}

	return result
}
