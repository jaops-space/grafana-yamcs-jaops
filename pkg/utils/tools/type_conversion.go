package tools

import (
	"fmt"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
)

func ConditionnalConverter(rawValue bool, returned *pvalue.ParameterValue) any {
	if rawValue {
		return ConvertProtobufValue(returned.RawValue)
	} else {
		return ConvertProtobufValue(returned.EngValue)
	}
}

func ConvertProtobufValue(returned *protobuf.Value) any {
	var res any
	switch *returned.Type {

	case protobuf.Value_FLOAT:
		res = returned.GetFloatValue()

	case protobuf.Value_DOUBLE:
		res = returned.GetDoubleValue()
	case protobuf.Value_UINT32:
		res = returned.GetUint32Value()
	case protobuf.Value_SINT32:
		res = returned.GetSint32Value()
	case protobuf.Value_BINARY:
		res = returned.GetBinaryValue()
	case protobuf.Value_STRING:
		res = returned.GetStringValue()
	case protobuf.Value_TIMESTAMP:
		res = returned.GetTimestampValue()
	case protobuf.Value_UINT64:
		res = returned.GetUint64Value()
	case protobuf.Value_SINT64:
		res = returned.GetSint64Value()
	case protobuf.Value_BOOLEAN:
		res = returned.GetBooleanValue()
	case protobuf.Value_AGGREGATE:
		res = returned.GetAggregateValue()
	case protobuf.Value_ARRAY:
		res = returned.GetArrayValue()

	case protobuf.Value_ENUMERATED:
		res = returned.GetStringValue()
	}

	return res
}

func StringifyValue(value *protobuf.Value) string {
	if value == nil || value.Type == nil {
		return "<nil>"
	}

	switch *value.Type {
	case protobuf.Value_FLOAT, protobuf.Value_DOUBLE:
		return fmt.Sprintf("%.2f", ConvertProtobufValue(value))
	case protobuf.Value_UINT32, protobuf.Value_SINT32, protobuf.Value_UINT64, protobuf.Value_SINT64:
		return fmt.Sprintf("%d", ConvertProtobufValue(value))
	case protobuf.Value_BINARY:
		return fmt.Sprintf("0x%x", ConvertProtobufValue(value))
	case protobuf.Value_TIMESTAMP:
		return time.Unix(value.GetTimestampValue(), 0).Format(time.RFC3339)
	case protobuf.Value_BOOLEAN:
		return fmt.Sprintf("%t", ConvertProtobufValue(value))
	case protobuf.Value_AGGREGATE:
		agg := value.GetAggregateValue()
		result := "{"
		first := true
		for i, key := range agg.GetName() {
			if !first {
				result += ", "
			}
			value := agg.GetValue()[i]
			result += fmt.Sprintf("%s: %s", key, StringifyValue(value))
			first = false
		}
		result += "}"
		return result
	case protobuf.Value_ARRAY:
		arr := value.GetArrayValue()
		result := "["
		for i, value := range arr {
			if i > 0 {
				result += ", "
			}
			result += StringifyValue(value)
		}
		result += "]"
		return result
	case protobuf.Value_STRING, protobuf.Value_ENUMERATED:
		return ConvertProtobufValue(value).(string)
	}

	return "<invalid value>"
}
