package plugin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
    "google.golang.org/protobuf/encoding/protojson"
    "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
    "github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

func DatasourceGraphFrame(querier *source.Querier, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	backend.Logger.Info("DatasourceGraphFrame called",
		"endpoint", q.EndpointID,
		"parameter", q.Parameter,
		"from", q.From,
		"to", q.To,
		"realtime", q.Realtime,
		"querier", querier != nil)

	yamcs := endpoint.GetClient()
	yamcs.SetSamplePointCount(q.MaxPoints)

	start := time.Unix(int64(q.From), 0)
	end := time.Unix(int64(q.To), 0)
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

	backend.Logger.Debug("Requesting parameter samples",
		"parameter", q.Parameter,
		"aggregatePath", aggregatePath,
		"startTime", start,
		"endTime", end,
		"yamcsFilter", q.YamcsFilter)

// 	// Convert YamcsFilter to source.YamcsFilterConfig
// 	var yamcsFilter *source.YamcsFilterConfig
// 	if q.YamcsFilter != nil {
// 		yamcsFilter = &source.YamcsFilterConfig{
// 			Enabled:   q.YamcsFilter.Enabled,
// 			Parameter: q.YamcsFilter.Parameter,
// 			Operator:  q.YamcsFilter.Operator,
// 			Value:     q.YamcsFilter.Value,
// 		}
// 	}

	if err != nil {
		backend.Logger.Error("Error requesting parameter samples", "error", err)
		return nil, err
	}

	pointCount := 0

	backend.Logger.Info("Received parameter samples",
		"parameter", q.Parameter,
		"aggregatePath", aggregatePath,
		"pointCount", pointCount)

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

func DatasourceSingleValueFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs := endpoint.GetClient()
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	// TODO: Pass filter parameters to YAMCS when server-side filtering is implemented
	lastValue, err := yamcs.GetParameterValueByName(endpoint.Instance, endpoint.Processor, q.Parameter)

	if err != nil {
		return nil, err
	}

	buffer := []client.ParameterValue{lastValue}

	frame := tools.ConvertBufferToFrame(buffer, q.Parameter+aggregatePath, false, false, aggregatePath, q.Realtime)
	SetUnitAndThresholds(endpoint, q.Parameter, frame)
	return frame, nil

}

func DatasourceDiscreteValueFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

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

	ranges, err := yamcs.GetParameterRangesByQueryWithTimeByNames(
		endpoint.Instance.GetName(),
		q.Parameter,
		map[string]string{
			"minRange":  minRange,
			"processor": endpoint.Processor.GetName(),
		},
		start,
		end,
	)

	if err != nil {
		return nil, err
	}

	frame := tools.ConvertRangesToFrame(ranges, q.Parameter+aggregatePath, aggregatePath)
	SetUnitAndThresholds(endpoint, q.Parameter, frame)
	return frame, nil

}

func DatasourceEventsFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

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
	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeTable}
	return frame, nil
}

func SetUnitAndThresholds(endpoint *source.YamcsEndpoint, parameter string, frame *data.Frame) {

	parameterDemand := endpoint.GetParameterDemand(parameter)

	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeGraph}

	for _, field := range frame.Fields {
		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.Unit = parameterDemand.Unit
		field.Config.Thresholds = &data.ThresholdsConfig{
			Mode:  data.ThresholdsModeAbsolute,
			Steps: make([]data.Threshold, 0, len(parameterDemand.Thresholds)),
		}
		for _, t := range parameterDemand.Thresholds {
			field.Config.Thresholds.Steps = append(field.Config.Thresholds.Steps, *t)
		}
	}
}

func DatasourceCommandFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

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

	return data.NewFrame(
		"command",
		data.NewField("info", nil, []json.RawMessage{commandRawJSON}),
		data.NewField("endpoint", nil, []string{q.EndpointID}),
	), nil
}

func DatasourceCommandHistoryFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

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
	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeTable}

	return frame, nil
}

func DatasourceTimeFrame(endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	return data.NewFrame("response",
		data.NewField("current_time", nil, []time.Time{endpoint.CurrentTime}),
	), nil
}

// telemetryPointsToSamples converts TelemetryPoint slice to Yamcs Sample slice.
// This enables compatibility with existing frame conversion tools.
func telemetryPointsToSamples(points []source.TelemetryPoint) []client.Sample {
	if len(points) == 0 {
		return []client.Sample{}
	}

	samples := make([]*pvalue.TimeSeries_Sample, len(points))
	for i, pt := range points {
		n := int32(1)
		samples[i] = &pvalue.TimeSeries_Sample{
			Time: timestamppb.New(pt.Time),
			Avg:  pt.Value,
			N:    &n,
		}
	}
	return samples
}