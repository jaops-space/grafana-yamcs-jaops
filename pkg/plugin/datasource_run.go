package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

func getStreamTickerInterval(q PluginQuery, fallback time.Duration) time.Duration {
	if q.MaxPoints <= 0 || q.To <= q.From {
		return minStreamTickerInterval(fallback)
	}

	timeWindow := time.Duration(q.To-q.From) * time.Second
	if timeWindow <= 0 {
		return minStreamTickerInterval(fallback)
	}

	interval := timeWindow / time.Duration(q.MaxPoints)
	interval = minStreamTickerInterval(interval)
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}

// livenessCheckInterval bounds how long an event-driven stream (alarms, links,
// command history) can go without checking whether the underlying yamcs
// websocket connection is still alive. Without this, a silently dropped
// connection (e.g. one that doesn't produce a new event to react to) leaves
// the stream blocked forever on its signal channel, never surfacing an error
// and never letting Grafana Live re-establish the stream.
const livenessCheckInterval = 5 * time.Second

func minStreamTickerInterval(interval time.Duration) time.Duration {
	if interval < 200*time.Millisecond {
		return 200 * time.Millisecond
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

	return minStreamTickerInterval(scaled)
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
	streamBenchmarkStats.recordRunStream(req.Path)

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

			started := time.Now()
			buffer := endpoint.GetAndClearParameterStreamBuffer(q.Parameter, req.Path)
			if len(buffer) == 0 {
				continue
			}

			if q.Type == DiscreteValue {
				frame := tools.ConvertDiscreteBufferToFrame(buffer, q.Parameter, q.AutomaticColors, false)
				sender.SendFrame(
					frame,
					data.IncludeDataOnly,
				)
				streamBenchmarkStats.recordRunStreamWork(req.Path, time.Since(started), len(buffer))
				continue
			}
			if q.Type == SingleValue {
				frame := tools.ConvertSingleValueBufferToFrame(buffer, q.Parameter, false)
				sender.SendFrame(
					frame,
					data.IncludeAll,
				)
				continue
			}

			average := len(buffer) > 3
			var frame *data.Frame
			if average {
				frame = tools.ConvertBufferToAverageFrame(buffer, q.Parameter, getMin, getMax, false)
			} else {
				frame = tools.ConvertBufferToFrame(buffer, q.Parameter, getMin, getMax, false)
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
			streamBenchmarkStats.recordRunStreamWork(req.Path, time.Since(started), len(buffer))
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
	if signal == nil {
		return backend.DownstreamErrorf("events stream signal not registered for path %q", req.Path)
	}

	defer endpoint.WithdrawEventsStreamRequest(req.Path)

	// Periodic liveness check, see livenessCheckInterval comment.
	ticker := time.NewTicker(livenessCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
		case event, ok := <-signal:
			if !ok {
				return nil
			}

			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			frame := tools.ConvertEventsToFrame(endpoint.DrainEventsSignal(event, signal))
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
	if err := endpoint.RequestCommandHistoryStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}
	signal := endpoint.GetCommandHistorySignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("command history stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawCommandHistoryStreamRequest(req.Path)

	// Periodic liveness check so a silently dropped websocket connection is
	// detected even when no new command history events are arriving, instead
	// of blocking forever on the signal channel.
	ticker := time.NewTicker(livenessCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
		case command, ok := <-signal:
			if !ok {
				return nil
			}
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
			frame := tools.ConvertCommandListToFrame(endpoint.DrainCommandHistorySignal(command, signal))
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
	if err := endpoint.RequestAlarmsStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}
	signal := endpoint.GetAlarmsSignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("alarms stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawAlarmsStreamRequest(req.Path)

	// Periodic liveness check, see livenessCheckInterval comment.
	ticker := time.NewTicker(livenessCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
		case _, ok := <-signal:
			if !ok {
				return nil
			}
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
			endpoint.DrainAlarmsSignal(signal)

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

	if err := endpoint.RequestLinksStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}

	signal := endpoint.GetLinksSignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("links stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawLinksStreamRequest(req.Path)

	// Periodic liveness check, see livenessCheckInterval comment.
	ticker := time.NewTicker(livenessCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}
		case link, ok := <-signal:
			if !ok {
				return nil
			}
			if !yamcs.IsWebSocketConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			linksEvents := endpoint.DrainLinksSignal(link, signal)
			latestLink := linksEvents[len(linksEvents)-1]

			frame, err := tools.ConvertLinksToFrame(latestLink.GetLinks())
			if err != nil {
				return err
			}

			backend.Logger.Debug(
				"sending links stream frame",
				"path", req.Path,
				"linkCount", len(latestLink.GetLinks()),
				"fieldCount", len(frame.Fields),
			)
			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
		}
	}
}
