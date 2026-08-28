package plugin

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
)

func RunSubscriptionStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	ctx, yamcs, cancel, err := beginStreamGuard(ctx, endpoint)
	if err != nil {
		return err
	}
	defer cancel()

	ticker := time.NewTicker(time.Second)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:

			subscriptions := make([]string, 0)

			for _, sub := range yamcs.ParameterSubscriptions {
				for param := range sub.ActiveSubscriptions {
					subscriptions = append(subscriptions, param)
				}
			}

			frame := data.NewFrame("response",
				data.NewField("parameter", nil, subscriptions),
			)

			sender.SendFrame(
				frame,
				data.IncludeAll,
			)

		}
	}

}

func RunDemandsStream(ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender,
	endpoint *source.YamcsEndpoint,
	q PluginQuery) error {

	// Unlike the other streams, the demands stream is a local diagnostic view
	// derived purely from in-memory endpoint state (which parameters have
	// active streams and when they last received a value) - it doesn't depend
	// on the WebSocket connection itself, so it intentionally keeps running
	// across disconnects/reconnects and only exits when the panel is closed
	// (ctx cancelled).
	ticker := time.NewTicker(time.Second)

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
				data.NewField("parameter", nil, parameters),
				data.NewField("stream_path", nil, streamPaths),
				data.NewField("last_value_at", nil, lastReceived),
			)

			sender.SendFrame(
				frame,
				data.IncludeAll,
			)

		}
	}

}
