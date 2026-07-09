package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
)

func DatasourceGraphFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	backend.Logger.Debug("DatasourceGraphFrame called",
		"endpoint", q.EndpointID,
		"parameter", q.Parameter,
		"from", q.From,
		"to", q.To,
		"realtime", q.Realtime)

	yamcs, err := endpoint.GetClient()

	// TODO: sample count as direct parameter of yamcs client
	yamcs.SetSamplePointCount(q.MaxPoints)

	start := time.Unix(int64(q.From), 0)
	end := time.Unix(int64(q.To), 0)

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

	// Include aggregatePath in the API call to get the correct value type (Position.X returns INTEGER instead of AGGREGATE)
	samples, err := yamcs.GetParameterSamplesInProcessorByNames(ctx, endpoint.GetInstanceName(), endpoint.GetProcessorName(), q.Parameter+aggregatePath, start, end)

	if err != nil {
		backend.Logger.Error("Error requesting parameter samples", "error", err)
		return nil, err
	}

	pointCount := 0

	backend.Logger.Debug("Received parameter samples",
		"parameter", q.Parameter,
		"aggregatePath", aggregatePath,
		"pointCount", pointCount)

	var getMin bool = false
	var getMax bool = false

	for _, getField := range q.Fields {
		getMin = getMin || (getField == "min")
		getMax = getMax || (getField == "max")
	}

	frame := tools.ConvertSampleBufferToFrame(samples, q.Parameter+aggregatePath, getMin, getMax)

	endpoint.SetUnitAndThresholds(ctx, q.Parameter, frame)
	return frame, nil
}

func DatasourceSingleValueFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	// TODO: Pass filter parameters to YAMCS when server-side filtering is implemented
	lastValue, err := yamcs.GetParameterValueByName(ctx, endpoint.GetInstanceName(), endpoint.GetProcessorName(), q.Parameter)

	if err != nil {
		return nil, err
	}

	buffer := []client.ParameterValue{lastValue}

	frame := tools.ConvertBufferToFrame(buffer, q.Parameter+aggregatePath, false, false, aggregatePath, false)
	endpoint.SetUnitAndThresholds(ctx, q.Parameter, frame)
	return frame, nil

}

func DatasourceDiscreteValueFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}

	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

	minRange := fmt.Sprint(int(end.Sub(start).Milliseconds()) / q.MaxPoints)

	ranges, err := yamcs.GetParameterRangesByQueryWithTimeByNames(
		ctx,
		endpoint.GetInstanceName(),
		q.Parameter,
		map[string]string{
			"minRange":  minRange,
			"processor": endpoint.GetProcessorName(),
		},
		start,
		end,
	)

	if err != nil {
		return nil, err
	}

	frame := tools.ConvertRangesToFrame(ranges, q.Parameter+aggregatePath, aggregatePath)
	endpoint.SetUnitAndThresholds(ctx, q.Parameter, frame)
	return frame, nil

}

func DatasourceEventsFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)

	iterator := yamcs.ListEventsWithinTimeRange(ctx, endpoint.GetInstanceName(), start, end)
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

func DatasourceCommandFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	command, err := yamcs.GetCommandInfo(ctx, endpoint.GetInstanceName(), q.Command)
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

func DatasourceCommandHistoryFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	start, end := time.Unix(int64(q.From), 0), time.Unix(int64(q.To), 0)
	iterator := yamcs.ListCommandsHistory(ctx, endpoint.GetInstanceName(), start, end)
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

func DatasourceTimeFrame(_ context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	currentTime, ok := endpoint.GetCurrentTimeIfFresh(15 * time.Second)
	if !ok {
		return data.NewFrame(
			"response",
			data.NewField("time", nil, []time.Time{}),
			data.NewField("speed", nil, []float64{}),
		), nil
	}
	replaySpeedMultiplier, err := endpoint.GetReplaySpeedMultiplier()
	if err != nil {
		return nil, err
	}

	return data.NewFrame("response",
		data.NewField("time", nil, []time.Time{currentTime}),
		data.NewField("speed", nil, []float64{replaySpeedMultiplier}),
	), nil
}

func DatasourceAlarmsFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	alarmList, err := yamcs.ListProcessorAlarms(ctx, endpoint.GetInstanceName(), endpoint.GetProcessorName())
	if err != nil {
		return nil, err
	}

	frame := tools.ConvertAlarmListToFrame(alarmList)
	frame.Meta = &data.FrameMeta{}
	frame.Meta.PreferredVisualization = data.VisTypeTable
	return frame, nil

}

func DatasourceLinksFrame(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery) (*data.Frame, error) {
	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, err
	}
	list, err := yamcs.ListLinks(ctx, endpoint.GetInstanceName())
	if err != nil {
		return nil, err
	}

	return buildLinksFrame(list)
}

func buildLinksFrame(items []*links.LinkInfo) (*data.Frame, error) {
	results := make([]LinkInfoResult, 0, len(items))
	for _, link := range items {
		results = append(results, convertLinkInfo(link))
	}

	payload, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	frame := data.NewFrame(
		"links",
		data.NewField("linksJson", nil, []string{string(payload)}),
	)
	frame.Meta = &data.FrameMeta{PreferredVisualization: data.VisTypeTable}

	return frame, nil
}
