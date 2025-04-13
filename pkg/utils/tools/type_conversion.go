package tools

import (
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
		res = returned.GetSint64Value()
	}

	return res
}
