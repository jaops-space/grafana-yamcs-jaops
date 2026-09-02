package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
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

// beginStreamGuard is the single place where every RunXStream handler checks
// that this endpoint's Yamcs WebSocket is connected before doing any real
// work, instead of each handler separately calling endpoint.GetClient(),
// checking IsWebSocketConnected() and wiring up its own
// `case <-yamcs.Disconnected()` arm.
//
// On success, it returns the endpoint's Yamcs client (for the rare handler
// that still needs it, e.g. RunSubscriptionStream) along with a context
// derived from ctx that is also cancelled - with cause
// backend.DownstreamErrorf("yamcs client disconnected") - the instant the
// connection is lost, so callers can rely solely on
// `case <-ctx.Done(): return context.Cause(ctx)` in their select loop for
// both panel-close and connection-loss, with no extra channel or re-check
// needed. The returned cancel func must be deferred by the caller to stop the
// small watcher goroutine once the stream ends.
//
// Not used by RunDemandsStream, which is intentionally WebSocket-independent
// (see its own doc comment).
func beginStreamGuard(ctx context.Context, endpoint *source.YamcsEndpoint) (context.Context, *client.YamcsClient, context.CancelFunc, error) {
	yamcs, err := endpoint.GetClient()
	if err != nil {
		return nil, nil, nil, backend.DownstreamError(err)
	}
	if !yamcs.IsWebSocketConnected() {
		return nil, nil, nil, backend.DownstreamErrorf("yamcs client disconnected")
	}

	streamCtx, cancel := context.WithCancelCause(ctx)
	go func() {
		select {
		case <-streamCtx.Done():
		case <-yamcs.Disconnected():
			cancel(backend.DownstreamErrorf("yamcs client disconnected"))
		}
	}()
	return streamCtx, yamcs, func() { cancel(nil) }, nil
}

func RunParameterStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

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
			return context.Cause(ctx)
		case <-ticker.C:

			started := time.Now()
			batch := endpoint.DrainParameterStream(q.Parameter, req.Path)
			if len(batch) == 0 {
				continue
			}

			if q.Type == DiscreteValue {
				frame := tools.ConvertDiscreteBufferToFrame(batch, q.Parameter, q.AutomaticColors, false)
				sender.SendFrame(
					frame,
					data.IncludeDataOnly,
				)
				streamBenchmarkStats.recordRunStreamWork(req.Path, time.Since(started), len(batch))
				continue
			}
			if q.Type == SingleValue {
				frame := tools.ConvertSingleValueBufferToFrame(batch, q.Parameter, false)
				sender.SendFrame(
					frame,
					data.IncludeAll,
				)
				continue
			}

			average := len(batch) > 3
			var frame *data.Frame
			if average {
				frame = tools.ConvertBufferToAverageFrame(batch, q.Parameter, getMin, getMax, false)
			} else {
				frame = tools.ConvertBufferToFrame(batch, q.Parameter, getMin, getMax, false)
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
			streamBenchmarkStats.recordRunStreamWork(req.Path, time.Since(started), len(batch))
		}
	}

}

func RunEventStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

	signal, err := endpoint.RequestEventsStream(ctx, req.Path)
	if err != nil {
		return backend.DownstreamError(err)
	}
	if signal == nil {
		return backend.DownstreamErrorf("events stream signal not registered for path %q", req.Path)
	}

	defer endpoint.WithdrawEventsStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case _, ok := <-signal:
			if !ok {
				return nil
			}

			batch := endpoint.DrainEventsStream(req.Path)
			if len(batch) == 0 {
				continue
			}
			frame := tools.ConvertEventsToFrame(batch)
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

	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

	// Start listening for command history entries for this path
	if err := endpoint.RequestCommandHistoryStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}
	signal := endpoint.GetCommandHistorySignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("command history stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawCommandHistoryStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case _, ok := <-signal:
			if !ok {
				return nil
			}
			batch := endpoint.DrainCommandHistoryStream(req.Path)
			if len(batch) == 0 {
				continue
			}
			frame := tools.ConvertCommandListToFrame(batch)
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

	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

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
			return context.Cause(ctx)
		case <-ticker.C:

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

	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

	// Start listening for alarm events for this path
	if err := endpoint.RequestAlarmsStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}
	signal := endpoint.GetAlarmsSignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("alarms stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawAlarmsStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case _, ok := <-signal:
			if !ok {
				return nil
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
	ctx, _, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

	if err := endpoint.RequestLinksStream(ctx, req.Path); err != nil {
		return backend.DownstreamError(err)
	}

	signal := endpoint.GetLinksSignal(req.Path)
	if signal == nil {
		return backend.DownstreamErrorf("links stream signal not registered for path %q", req.Path)
	}
	defer endpoint.WithdrawLinksStreamRequest(req.Path)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case _, ok := <-signal:
			if !ok {
				return nil
			}

			linksEvents := endpoint.DrainLinksStream(req.Path)
			if len(linksEvents) == 0 {
				// The coalesced notify channel can fire more times than
				// there are distinct batches of new data (e.g. two pushes
				// arriving close together each queue a notify, but the
				// first drain already picked up both) - just skip this
				// wakeup rather than indexing into an empty slice.
				continue
			}
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
