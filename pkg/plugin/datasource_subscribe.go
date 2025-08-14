package plugin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/multiplexer"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"google.golang.org/protobuf/encoding/protojson"
)

func DatasourceGraphFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	yamcs.SetSamplePointCount(q.MaxPoints)
	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	offset := time.Since(endpoint.CurrentTime)
	if q.Realtime {
		end = time.Now()
		start = start.Add(-offset)
		end = end.Add(-offset)
	}
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	backend.Logger.Debug("Requesting parameter samples", "parameter", q.Parameter, "aggregatePath", aggregatePath)
	samples, err := yamcs.GetParameterSamplesByNames(endpoint.Instance, q.Parameter+aggregatePath, start, end)
	if err != nil {
		backend.Logger.Error("Error requesting parameter samples", "error", err)
		return nil, err
	}

	backend.Logger.Debug("Received parameter samples", "parameter", q.Parameter, "aggregatePath", aggregatePath)

	var getMin bool = false
	var getMax bool = false
	for _, getField := range q.Fields {
		getMin = getMin || (getField == "min")
		getMax = getMax || (getField == "max")
	}

	var frame *data.Frame
	if q.Realtime {
		frame = tools.ConvertSampleBufferToFrameWithOffset(samples, q.Parameter+aggregatePath, getMin, getMax, offset)
	} else {
		frame = tools.ConvertSampleBufferToFrame(samples, q.Parameter+aggregatePath, getMin, getMax)
	}

	SetUnitAndThresholds(endpoint, q.Parameter, frame)
	return frame, nil

}

func DatasourceSingleValueFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	lastValue, err := yamcs.GetParameterValueByName(endpoint.Instance, endpoint.Processor, q.Parameter)

	if err != nil {
		return nil, err
	}

	buffer := []client.ParameterValue{lastValue}

	frame := tools.ConvertBufferToFrame(buffer, q.Parameter+aggregatePath, false, false, aggregatePath, q.Realtime)
	SetUnitAndThresholds(endpoint, q.Parameter, frame)
	return frame, nil

}

func DatasourceDiscreteValueFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()

	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	if q.Realtime {
		end = time.Now()
		offset := time.Since(endpoint.CurrentTime)
		start = start.Add(-offset)
		end = end.Add(-offset)
	}
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	minRange := fmt.Sprint(int(end.Sub(start).Milliseconds()) / q.MaxPoints)

	ranges, err := yamcs.GetParameterRangesByQueryWithTimeByNames(endpoint.Instance.GetName(), q.Parameter, map[string]string{
		"minRange":  minRange,
		"processor": endpoint.Processor.GetName(),
	}, start, end)

	if err != nil {
		return nil, err
	}

	frame := tools.ConvertRangesToFrame(ranges, q.Parameter+aggregatePath, aggregatePath)
	SetUnitAndThresholds(endpoint, q.Parameter, frame)
	return frame, nil

}

func DatasourceEventsFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	if q.Realtime {
		end = time.Now()
		offset := time.Since(endpoint.CurrentTime)
		start = start.Add(-offset)
		end = end.Add(-offset)
	}

	iterator := yamcs.ListEventsWithinTimeRange(endpoint.Instance, start, end)
	events := []*events.Event{}
	for iterator.HasNext() {
		currentEvents, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		events = append(events, currentEvents...)
	}
	frame := tools.ConvertEventsToFrame(events)
	frame.Meta = &data.FrameMeta{}
	frame.Meta.PreferredVisualization = data.VisTypeTable
	return frame, nil

}

func SetUnitAndThresholds(endpoint *multiplexer.YamcsEndpoint, parameter string, frame *data.Frame) {

	parameterDemand := endpoint.GetParameterDemand(parameter)

	frame.Meta = &data.FrameMeta{}
	frame.Meta.PreferredVisualization = data.VisTypeGraph
	for _, field := range frame.Fields {
		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.Unit = parameterDemand.Unit
		field.Config.Thresholds = &data.ThresholdsConfig{}
		field.Config.Thresholds.Mode = data.ThresholdsModeAbsolute
		field.Config.Thresholds.Steps = make([]data.Threshold, 0)
		for _, threshold := range parameterDemand.Thresholds {
			field.Config.Thresholds.Steps = append(field.Config.Thresholds.Steps, *threshold)
		}
	}

}

func DatasourceCommandFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	command, err := yamcs.GetCommandInfo(endpoint.Instance, q.Command)
	if err != nil {
		return nil, err
	}
	commandJSON, err := protojson.Marshal(command)
	if err != nil {
		return nil, err
	}
	commandRawJSON := json.RawMessage(commandJSON)

	frame := data.NewFrame("command",
		data.NewField("info", nil, []json.RawMessage{commandRawJSON}),
		data.NewField("endpoint", nil, []string{q.EndpointID}))
	return frame, nil

}

func DatasourceCommandHistoryFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	iterator := yamcs.ListCommandsHistory(endpoint.Instance, start, end)
	commandList := make([]*commanding.CommandHistoryEntry, 0)
	for iterator.HasNext() {
		commands, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		commandList = append(commandList, commands...)
	}

	frame := tools.ConvertCommandListToFrame(commandList)
	frame.Meta = &data.FrameMeta{}
	frame.Meta.PreferredVisualization = data.VisTypeTable
	return frame, nil

}

func DatasourceTimeFrame(endpoint *multiplexer.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	frame := data.NewFrame("response", data.NewField("current_time", nil, []time.Time{endpoint.CurrentTime}))
	return frame, nil

}
