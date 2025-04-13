package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/multiplexer"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
)

func RunParameterStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *multiplexer.YamcsEndpoint,
	q PluginQuery) error {

	yamcs := endpoint.GetClient()

	backend.Logger.Debug("Requesting parameter stream", "parameter", q.Parameter, "path", req.Path)
	err := endpoint.RequestNewParameterStream(q.Parameter, req.Path)
	if err != nil {
		backend.Logger.Error("Error requesting parameter stream", "error", err)
		return err
	}
	backend.Logger.Debug("Requested parameter stream", "parameter", q.Parameter, "path", req.Path)
	defer endpoint.WithdrawParameterStreamRequest(q.Parameter, req.Path)

	timeWindow := time.Duration(q.To-q.From) * time.Second
	tickerInterval := timeWindow / time.Duration(q.MaxPoints)

	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	aggregatePath := ""
	if len(q.AggregatePath) > 0 {
		aggregatePath = "." + q.AggregatePath
	}

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
				return backend.DownstreamErrorf("Yamcs Client is disconnected")
			}

			buffer := endpoint.GetParameterStreamBuffer(q.Parameter, req.Path)
			if len(buffer) == 0 {
				continue
			}

			average := len(buffer) > 3
			var frame *data.Frame
			if average {
				frame = tools.ConvertBufferToAverageFrame(buffer, q.Parameter+aggregatePath, getMin, getMax, aggregatePath)
			} else {
				frame = tools.ConvertBufferToFrame(buffer, q.Parameter+aggregatePath, getMin, getMax, aggregatePath)
			}

			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)

			endpoint.ClearParameterStream(q.Parameter, req.Path)

		}
	}

}

func RunEventStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *multiplexer.YamcsEndpoint,
	q PluginQuery) error {

	yamcs := endpoint.GetClient()

	endpoint.RequestEventsStream(req.Path)

	timeWindow := time.Duration(q.To-q.From) * time.Second
	tickerInterval := timeWindow / time.Duration(q.MaxPoints)
	ticker := time.NewTicker(tickerInterval)

	defer ticker.Stop()
	defer endpoint.ClearEventsStream(req.Path)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			if !yamcs.WebSocket.IsConnected() {
				return backend.DownstreamErrorf("yamcs client disconnected")
			}

			buffer := endpoint.GetEventsStream(req.Path)
			if len(buffer) == 0 {
				continue
			}
			frame := tools.ConvertEventsToFrame(buffer)
			sender.SendFrame(
				frame,
				data.IncludeDataOnly,
			)
			endpoint.ClearEventsStream(req.Path)

		}
	}

}

func RunDemandsStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *multiplexer.YamcsEndpoint,
	q PluginQuery) error {

	tickerInterval := 1 * time.Second
	ticker := time.NewTicker(tickerInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			streamPaths := make([]string, 0)
			parameters := make([]string, 0)
			lastReceived := make([]time.Time, 0)

			for _, parameter := range endpoint.Parameters {
				for _, stream := range parameter.Streams {
					streamPaths = append(streamPaths, stream.Path)
					parameters = append(parameters, parameter.Name)
					lastReceived = append(lastReceived, parameter.LastReceived)
				}
			}

			frame := data.NewFrame("response",
				data.NewField("Parameter", nil, parameters),
				data.NewField("Stream Path", nil, streamPaths),
				data.NewField("Last Value Received", nil, lastReceived),
			)

			sender.SendFrame(
				frame,
				data.IncludeAll,
			)

		}
	}

}

func RunSubscriptionStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *multiplexer.YamcsEndpoint,
	q PluginQuery) error {

	tickerInterval := 1 * time.Second
	ticker := time.NewTicker(tickerInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:

			subscriptions := make([]string, 0)
			yamcs := endpoint.GetClient()
			if yamcs == nil {
				return backend.DownstreamErrorf("No client found")
			}

			for _, sub := range yamcs.ParameterSubscriptions {
				for param := range sub.ActiveSubscriptions {
					subscriptions = append(subscriptions, param)
				}
			}

			frame := data.NewFrame("response",
				data.NewField("Parameter", nil, subscriptions),
			)

			sender.SendFrame(
				frame,
				data.IncludeAll,
			)

		}
	}

}
