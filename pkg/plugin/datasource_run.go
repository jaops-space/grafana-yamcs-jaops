package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/events"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

func getStreamTickerInterval(q PluginQuery, fallback time.Duration) time.Duration {
	if q.MaxPoints <= 0 || q.To <= q.From {
		return fallback
	}

	timeWindow := time.Duration(q.To-q.From) * time.Second
	if timeWindow <= 0 {
		return fallback
	}

	interval := timeWindow / time.Duration(q.MaxPoints)
	if interval < 200*time.Millisecond {
		return 200 * time.Millisecond
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}

func scaleTickerIntervalByReplay(endpoint *source.YamcsEndpoint, baseInterval time.Duration) time.Duration {
	if baseInterval <= 0 {
		baseInterval = time.Second
	}

	multiplier, err := endpoint.GetReplaySpeedMultiplier()
	if err != nil {
		backend.Logger.Error("could not retreive processor replay speed", "error", err)
		return 1
	}
	if multiplier <= 1 {
		return baseInterval
	}

	scaled := time.Duration(float64(baseInterval) / multiplier)

	return scaled
}

func RunParameterStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	if !yamcs.IsWebSocketConnected() {
		yamcs.EstablishWebSocketConnection(ctx)
	}

	backend.Logger.Debug("Requesting parameter stream", "parameter", q.Parameter, "path", req.Path)
	err = endpoint.RequestNewParameterStream(ctx, q.Parameter, req.Path)
	if err != nil {
		backend.Logger.Error("Error requesting parameter stream", "error", err)
		return err
	}
	backend.Logger.Debug("Requested parameter stream", "parameter", q.Parameter, "path", req.Path)
	defer endpoint.WithdrawParameterStreamRequest(ctx, q.Parameter, req.Path)

	tickerInterval := getStreamTickerInterval(q, time.Second)
	tickerInterval = scaleTickerIntervalByReplay(endpoint, tickerInterval)

	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	var getMin bool = false
	var getMax bool = false
	for _, getField := range q.Fields {
		getMin = getMin || (getField == "min")
		getMax = getMax || (getField == "max")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			buffer := endpoint.GetAndClearParameterStreamBuffer(q.Parameter, req.Path)
			if len(buffer) == 0 {
				continue
			}

			average := len(buffer) > 3
			var frame *data.Frame
			if average {
				frame = tools.ConvertBufferToAverageFrame(buffer, q.Parameter, getMin, getMax, "", false)
			} else {
				frame = tools.ConvertBufferToFrame(buffer, q.Parameter, getMin, getMax, "", false)
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}

}

func RunEventStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	if !yamcs.IsWebSocketConnected() {
		return backend.DownstreamErrorf("yamcs client disconnected")
	}
	signal, err := endpoint.RequestEventsStream(ctx, req.Path)
	if err != nil {
		return backend.DownstreamError(err)
	}

	defer endpoint.WithdrawEventsStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-signal:

			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			frame := tools.ConvertEventsToFrame([]*events.Event{event})
			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}

}

func RunCommandHistoryStream(
	ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery,
) error {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	// Start listening for command history entries for this path
	endpoint.RequestCommandHistoryStream(ctx, req.Path)
	signal := endpoint.GetCommandHistorySignal(req.Path)
	defer endpoint.WithdrawCommandHistoryStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-signal:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
			// TODO: remove array overhead
			frame := tools.ConvertCommandListToFrame([]*commanding.CommandHistoryEntry{command})
			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}
}

func RunTimeStream(
	ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery,
) error {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	if !yamcs.IsWebSocketConnected() {
		return backend.DownstreamErrorf("yamcs client disconnected")
	}

	err = endpoint.RequestTime(ctx)
	if err != nil {
		return backend.DownstreamError(err)
	}

	// Calculate ticker interval
	tickerInterval := scaleTickerIntervalByReplay(endpoint, time.Second)
	ticker := time.NewTicker(tickerInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			currentTime, ok := endpoint.GetCurrentTimeIfFresh(15 * time.Second)
			if !ok {
				continue
			}
			replaySpeedMultiplier, err := endpoint.GetReplaySpeedMultiplier()
			if err != nil {
				return backend.DownstreamError(err)
			}

			frame := data.NewFrame("response",
				data.NewField("time", nil, []time.Time{currentTime}),
				data.NewField("speed", nil, []float64{replaySpeedMultiplier}),
			)

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}
}

func RunAlarmsStream(
	ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery,
) error {

	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	if !yamcs.IsWebSocketConnected() {
		return backend.DownstreamErrorf("yamcs client disconnected")
	}

	// Start listening for alarm events for this path
	endpoint.RequestAlarmsStream(ctx, req.Path)
	signal := endpoint.GetAlarmsSignal(req.Path)
	defer endpoint.WithdrawAlarmsStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-signal:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			buffer := endpoint.GetAlarmsStream(req.Path)
			frame := tools.ConvertAlarmListToFrame(buffer)

			globalAlarmStatus := endpoint.GetGlobalAlarmStatus()
			if globalAlarmStatus != nil {
				globalStatus := map[string]interface{}{
					"unacknowledgedCount":    globalAlarmStatus.GetUnacknowledgedCount(),
					"unacknowledgedActive":   globalAlarmStatus.GetUnacknowledgedActive(),
					"unacknowledgedSeverity": globalAlarmStatus.GetUnacknowledgedSeverity().String(),
					"acknowledgedCount":      globalAlarmStatus.GetAcknowledgedCount(),
					"acknowledgedActive":     globalAlarmStatus.GetAcknowledgedActive(),
					"acknowledgedSeverity":   globalAlarmStatus.GetAcknowledgedSeverity().String(),
					"shelvedCount":           globalAlarmStatus.GetShelvedCount(),
					"shelvedActive":          globalAlarmStatus.GetShelvedActive(),
					"shelvedSeverity":        globalAlarmStatus.GetShelvedSeverity().String(),
				}

				frame.Meta = &data.FrameMeta{
					Custom: map[string]interface{}{
						"globalAlarmStatus": globalStatus,
					},
				}
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)

			endpoint.ClearAlarmsStream(req.Path)
		}
	}
}

func RunLinksStream(
	ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery,
) error {
	yamcs, err := endpoint.GetClient()
	if err != nil {
		return backend.DownstreamError(err)
	}

	if !yamcs.IsWebSocketConnected() {
		return backend.DownstreamErrorf("yamcs client disconnected")
	}

	endpoint.RequestLinksStream(ctx, req.Path)

	signal := endpoint.GetLinksSignal(req.Path)
	defer endpoint.WithdrawLinksStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case link := <-signal:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			frame, err := buildLinksFrame(link.GetLinks())
			if err != nil {
				return err
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}
}
