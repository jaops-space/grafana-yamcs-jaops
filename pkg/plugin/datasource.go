package plugin

import (
	"context"
	"encoding/json"

	"github.com/gorilla/mux"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

// NewDatasource creates a new Datasource instance. Each instance gets its own
// Multiplexer so that different datasource configurations do not collide.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {

	var datasource Datasource

	cfg, secure, err := config.ExtractConfig(settings)
	if err != nil {
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	// Create a per-instance multiplexer so different datasource instances
	// do not share state or collide with each other.
	multiplexer, err := source.NewMultiplexer(cfg, secure)
	if err != nil {
		return nil, err
	}
	datasource.multiplexer = multiplexer

	multiplexer.Connect(ctx, true)

	router := mux.NewRouter()
	datasource.registerRoutes(router)
	datasource.CallResourceHandler = httpadapter.New(router)

	return &datasource, nil

}

// SubscribeStream handles the initial data request when a user subscribes to a stream.
// It fetches the historical data based on the query and returns it as the initial response.
func (d *Datasource) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {

	d.multiplexer.Connect(ctx, true)

	var q PluginQuery

	// Parse the query from the request payload
	if err := json.Unmarshal(req.Data, &q); err != nil {
		return nil, err
	}

	// Retrieve the endpoint associated with the requested stream
	endpoint, err := d.multiplexer.GetEndpoint(q.EndpointID)
	if err != nil {
		return nil, err
	}

	// Create a Grafana data frame based on the requested query type
	var frame *data.Frame
	switch q.Type {
	case Graph:
		frame, err = DatasourceGraphFrame(ctx, endpoint, q)
	case SingleValue, Image:
		frame, err = DatasourceSingleValueFrame(ctx, endpoint, q)
	case DiscreteValue:
		frame, err = DatasourceDiscreteValueFrame(ctx, endpoint, q)
	case Events:
		frame, err = DatasourceEventsFrame(ctx, endpoint, q)
	case Commanding:
		frame, err = DatasourceCommandFrame(ctx, endpoint, q)
	case CommandHistory:
		frame, err = DatasourceCommandHistoryFrame(ctx, endpoint, q)
	case Alarms:
		frame, err = DatasourceAlarmsFrame(ctx, endpoint, q)
	case Links:
		frame, err = DatasourceLinksFrame(ctx, endpoint, q)
	case Demands, Subscriptions:
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusOK,
		}, nil
	case Time:
		frame, err = DatasourceTimeFrame(ctx, endpoint, q)
	default:
		return nil, exception.New("query type not identified", "QUERY_TYPE_NOT_FOUND")
	}

	if err != nil {
		return nil, err
	}

	// Convert the data frame into an initial response format for Grafana
	initialData, err := backend.NewInitialFrame(frame, data.IncludeAll)
	if err != nil {
		return nil, err
	}

	return &backend.SubscribeStreamResponse{
		Status:      backend.SubscribeStreamStatusOK,
		InitialData: initialData,
	}, nil
}

// PublishStream is required by the plugin SDK but is not currently used.
// It simply returns an OK status.
func (d *Datasource) PublishStream(context.Context, *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusOK,
	}, nil
}

// RunStream continuously streams real-time data to users viewing the stream.
//
// Unlike a shared stream, this stream is user-specific because the data depends on
// user-configurable parameters such as time interval and number of data points.
//
// Graph stream frequency is determined by `timeInterval / maxDataPoints`, with
// a minimum interval of 200ms.
//
// If the parameter stream buffer has accumulated too much data, rather than sending
// every single data point, it calculates an average (for numeric values) or the most
// frequent value (for non-numeric values). This behavior ensures consistency with
// how historical data is retrieved, making real-time and historical views seamless.
func (d *Datasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {

	d.multiplexer.Connect(ctx, true)

	var q PluginQuery

	// Parse the query from the request payload
	if err := json.Unmarshal(req.Data, &q); err != nil {
		return err
	}

	// Retrieve the endpoint associated with the requested stream
	endpoint, err := d.multiplexer.GetEndpoint(q.EndpointID)
	if err != nil {
		return err
	}

	// Route the stream to the appropriate handler
	switch q.Type {
	case Graph, SingleValue, DiscreteValue, Image:
		return RunParameterStream(ctx, req, sender, endpoint, q)
	case Events:
		return RunEventStream(ctx, req, sender, endpoint, q)
	case Demands:
		return RunDemandsStream(ctx, req, sender, endpoint, q)
	case Subscriptions:
		return RunSubscriptionStream(ctx, req, sender, endpoint, q)
	case CommandHistory:
		return RunCommandHistoryStream(ctx, req, sender, endpoint, q)
	case Alarms:
		return RunAlarmsStream(ctx, req, sender, endpoint, q)
	case Links:
		return RunLinksStream(ctx, req, sender, endpoint, q)
	case Time:
		return RunTimeStream(ctx, req, sender, endpoint, q)
	default:
		return exception.New("query type not identified", "QUERY_TYPE_NOT_FOUND")
	}
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance is created.
func (d *Datasource) Dispose() {
	d.multiplexer.Dispose()
}
