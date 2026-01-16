package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/mux"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/multiplexer"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

// NewApp creates a new example *App instance.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {

	var datasource Datasource
	datasource.multiplexer = GlobalMultiplexer

	config, secure, err := config.ExtractConfig(settings)
	if err != nil {
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	router := mux.NewRouter()
	datasource.registerRoutes(router)
	datasource.CallResourceHandler = httpadapter.New(router)
	GlobalMultiplexer.Configuration = config
	GlobalMultiplexer.Secure = secure
	datasource.multiplexer = GlobalMultiplexer

	return &datasource, nil

}

// SubscribeStream handles the initial data request when a user subscribes to a stream.
// It fetches the historical data based on the query and returns it as the initial response.
func (d *Datasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
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
		frame, err = DatasourceGraphFrame(endpoint, q)
	case SingleValue, Image:
		frame, err = DatasourceSingleValueFrame(endpoint, q)
	case DiscreteValue:
		frame, err = DatasourceDiscreteValueFrame(endpoint, q)
	case Events:
		frame, err = DatasourceEventsFrame(endpoint, q)
	case Commanding:
		frame, err = DatasourceCommandFrame(endpoint, q)
	case CommandHistory:
		frame, err = DatasourceCommandHistoryFrame(endpoint, q)
	case Demands, Subscriptions:
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusOK,
		}, nil
	case Time:
		frame, err = DatasourceTimeFrame(endpoint, q)
	default:
		return nil, exception.New("Query type not identified", "QUERY_TYPE_NOT_FOUND")
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
// The streaming frequency is determined by: `timeInterval / maxDataPoints`.
//
// If the parameter stream buffer has accumulated too much data, rather than sending
// every single data point, it calculates an average (for numeric values) or the most
// frequent value (for non-numeric values). This behavior ensures consistency with
// how historical data is retrieved, making real-time and historical views seamless.
func (d *Datasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	var q PluginQuery

	// Parse the query from the request payload
	if err := json.Unmarshal(req.Data, &q); err != nil {
		return err
	}

	// Retrieve the endpoint associated with the requested stream
	endpoint, err := d.multiplexer.GetEndpoint(q.EndpointID)
	endpoint.RequestTime()
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
	case Time:
		return RunTimeStream(ctx, req, sender, endpoint, q)
	default:
		return nil
	}
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (d *Datasource) Dispose() {
	GlobalMultiplexer.Dispose()
}

// CheckHealth implements backend.CheckHealthHandler.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {

	settings := req.PluginContext.DataSourceInstanceSettings

	config, secure, err := config.ExtractConfig(*settings)
	if err != nil {
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	// Verify configuration
	if err := config.Validate(); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Invalid configuration: " + err.Error(),
		}, nil
	}

	testMux := multiplexer.NewMultiplexer(config)
	testMux.Secure = secure

	statusDetails := make(map[string]interface{})
	hostStatuses := make(map[string]string)
	endpointStatuses := make(map[string]string)
	hasErrors := false
	var errorMessages []string

	// Test all hosts
	backend.Logger.Debug("Testing Host Connectivity")
	for hostID := range config.Hosts {
		err := testMux.SetupHost(hostID)
		hostName := config.Hosts[hostID].Name
		displayName := hostName
		if displayName == "" {
			displayName = "Unknown Host"
		}

		status := "OK"
		if err != nil {
			status = err.Error()
			hasErrors = true
			errorMessages = append(errorMessages, fmt.Sprintf("Host '%s': %s", displayName, err.Error()))
		}

		hostStatuses[displayName] = status
	}

	// Test all endpoints - verify they can connect to their respective hosts and retrieve instance/processor info
	backend.Logger.Debug("Testing Endpoint Connectivity")
	for endpointID, endpointConfig := range config.Endpoints {
		endpointName := endpointConfig.Name
		if endpointName == "" {
			endpointName = endpointID
		}
		displayName := endpointName

		// Verify the endpoint's host was successfully set up
		hostConfig := config.Hosts[endpointConfig.Host]
		hostDisplayName := hostConfig.Name
		if hostDisplayName == "" {
			hostDisplayName = "Unknown Host"
		}

		hostStatus, hostExists := hostStatuses[hostDisplayName]
		if !hostExists {
			endpointStatuses[displayName] = "Host configuration not found"
			hasErrors = true
			errorMessages = append(errorMessages, fmt.Sprintf("Endpoint '%s': Host configuration not found", displayName))
			continue
		}

		if hostStatus != "OK" {
			endpointStatuses[displayName] = "Host connection failed: " + hostStatus
			hasErrors = true
			errorMessages = append(errorMessages, fmt.Sprintf("Endpoint '%s': Host connection failed: %s", displayName, hostStatus))
			continue
		}

		// Try to retrieve the endpoint to verify instance and processor exist
		_, err := testMux.GetEndpoint(endpointID)
		status := "OK"
		if err != nil {
			status = err.Error()
			hasErrors = true
			errorMessages = append(errorMessages, fmt.Sprintf("Endpoint '%s': %s", displayName, err.Error()))
		}

		endpointStatuses[displayName] = status
	}

	// Build detailed status response
	statusDetails["hosts"] = hostStatuses
	statusDetails["endpoints"] = endpointStatuses
	statusDetails["totalHosts"] = len(config.Hosts)
	statusDetails["totalEndpoints"] = len(config.Endpoints)

	jsonBytes, err := json.Marshal(statusDetails)
	if err != nil {
		return nil, err
	}

	// If any host or endpoint has errors, return error status
	if hasErrors {
		testMux.Dispose()
		return &backend.CheckHealthResult{
			Status:      backend.HealthStatusError,
			Message:     strings.Join(errorMessages, "\n"),
			JSONDetails: jsonBytes,
		}, nil
	}

	testMux.Dispose()
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		Message:     "Successfully connected to all Yamcs hosts and endpoints. Plugin is ready to use.",
		JSONDetails: jsonBytes,
	}, nil
}
