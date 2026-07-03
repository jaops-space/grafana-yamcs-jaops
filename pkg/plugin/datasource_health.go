package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

type ItemStatus struct {
	Status  string `json:"status"` // ok, warning, error
	Message string `json:"message,omitempty"`
}

type HealthDetails struct {
	Hosts          map[string]ItemStatus `json:"hosts"`
	Endpoints      map[string]ItemStatus `json:"endpoints"`
	TotalHosts     int                   `json:"totalHosts"`
	TotalEndpoints int                   `json:"totalEndpoints"`

	WarningHosts     []string `json:"warningHosts,omitempty"`
	WarningEndpoints []string `json:"warningEndpoints,omitempty"`
	ErrorHosts       []string `json:"errorHosts,omitempty"`
	ErrorEndpoints   []string `json:"errorEndpoints,omitempty"`
}

func okStatus() ItemStatus {
	return ItemStatus{Status: "ok", Message: "operational: no errors"}
}

func warningStatus(message string) ItemStatus {
	return ItemStatus{Status: "warning", Message: message}
}

func errorStatus(message string) ItemStatus {
	return ItemStatus{Status: "error", Message: message}
}

func (d *Datasource) validateConfigItems(y *config.YamcsPluginConfiguration) (*HealthDetails, bool, error) {
	details := &HealthDetails{
		Hosts:     map[string]ItemStatus{},
		Endpoints: map[string]ItemStatus{},
	}

	if y == nil {
		return details, true, fmt.Errorf("configuration is nil")
	}

	details.TotalHosts = len(y.Hosts)
	details.TotalEndpoints = len(y.Endpoints)

	hasValidationErrors := false

	if y.Hosts == nil {
		return details, true, fmt.Errorf("hosts configuration is missing")
	}

	if y.Endpoints == nil {
		return details, true, fmt.Errorf("endpoints configuration is missing")
	}

	if len(y.Hosts) == 0 {
		return details, true, fmt.Errorf("no hosts configured")
	}

	if len(y.Endpoints) == 0 {
		return details, true, fmt.Errorf("no endpoints configured")
	}

	for hostID, host := range y.Hosts {
		details.Hosts[hostID] = okStatus()

		if err := host.Validate(y); err != nil {
			details.Hosts[hostID] = errorStatus(err.Error())
			details.ErrorHosts = append(details.ErrorHosts, hostDisplayName(hostID, host))
			hasValidationErrors = true
		}
	}

	for endpointID, endpoint := range y.Endpoints {
		details.Endpoints[endpointID] = okStatus()

		if err := endpoint.Validate(y); err != nil {
			details.Endpoints[endpointID] = errorStatus(err.Error())
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpoint))
			hasValidationErrors = true
			continue
		}

		if endpoint.Host == "" {
			details.Endpoints[endpointID] = errorStatus("No host selected")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpoint))
			hasValidationErrors = true
			continue
		}

		if _, exists := y.Hosts[endpoint.Host]; !exists {
			details.Endpoints[endpointID] = errorStatus(fmt.Sprintf("References unknown host '%s'", endpoint.Host))
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpoint))
			hasValidationErrors = true
			continue
		}

		if hostStatus, exists := details.Hosts[endpoint.Host]; exists && hostStatus.Status == "error" {
			details.Endpoints[endpointID] = errorStatus("Host configuration is invalid")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpoint))
			hasValidationErrors = true
			continue
		}
	}

	return details, hasValidationErrors, nil
}

func (d *Datasource) applyConnectivityChecks(
	y *config.YamcsPluginConfiguration,
	secure *config.YamcsSecureConfiguration,
	details *HealthDetails,
) {
	testMux := source.NewMultiplexer(y)
	testMux.Secure = secure

	backend.Logger.Debug("Testing Host Connectivity")

	for hostID, hostConfig := range y.Hosts {
		if details.Hosts[hostID].Status != "ok" {
			continue
		}

		if err := testMux.SetupHost(hostID); err != nil {
			details.Hosts[hostID] = errorStatus(err.Error())
			details.ErrorHosts = append(details.ErrorHosts, hostDisplayName(hostID, hostConfig))
		}
	}

	backend.Logger.Debug("Testing Endpoint Connectivity")

	for endpointID, endpointConfig := range y.Endpoints {
		if details.Endpoints[endpointID].Status != "ok" {
			continue
		}

		hostStatus, hostExists := details.Hosts[endpointConfig.Host]
		if !hostExists {
			details.Endpoints[endpointID] = errorStatus("Host configuration not found")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpointConfig))
			continue
		}

		if hostStatus.Status == "error" || hostStatus.Status == "warning" {
			details.Endpoints[endpointID] = warningStatus("Skipped because host is unavailable")
			details.WarningEndpoints = append(details.WarningEndpoints, endpointDisplayName(endpointID, endpointConfig))
			continue
		}

		if _, err := testMux.GetEndpoint(endpointID); err != nil {
			details.Endpoints[endpointID] = errorStatus(err.Error())
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpointDisplayName(endpointID, endpointConfig))
		}
	}

	testMux.Dispose()
}

func (d *Datasource) evaluateHealth(
	y *config.YamcsPluginConfiguration,
	secure *config.YamcsSecureConfiguration,
) (backend.HealthStatus, string, json.RawMessage, error) {
	details, hasValidationErrors, err := d.validateConfigItems(y)
	if err != nil {
		jsonBytes, marshalErr := json.Marshal(details)
		if marshalErr != nil {
			return backend.HealthStatusError, "", nil, marshalErr
		}

		return backend.HealthStatusError, err.Error(), jsonBytes, nil
	}

	if !hasValidationErrors {
		d.applyConnectivityChecks(y, secure, details)
	}

	jsonBytes, err := json.Marshal(details)
	if err != nil {
		return backend.HealthStatusError, "", nil, err
	}

	status, message := buildHealthSummary(details)

	return status, message, jsonBytes, nil
}

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings := req.PluginContext.DataSourceInstanceSettings

	cfg, secure, err := config.ExtractConfig(*settings)
	if err != nil {
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	status, message, jsonDetails, err := d.evaluateHealth(cfg, secure)
	if err != nil {
		return nil, err
	}

	d.healthMutex.Lock()
	d.lastHealthDetails = jsonDetails
	d.healthMutex.Unlock()

	// save details and return only status and message
	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

func buildHealthSummary(details *HealthDetails) (backend.HealthStatus, string) {
	if len(details.ErrorHosts) > 0 || len(details.ErrorEndpoints) > 0 {
		var parts []string

		if len(details.ErrorHosts) > 0 {
			parts = append(parts, "hosts "+strings.Join(details.ErrorHosts, ", "))
		}

		if len(details.ErrorEndpoints) > 0 {
			parts = append(parts, "endpoints "+strings.Join(details.ErrorEndpoints, ", "))
		}

		return backend.HealthStatusError, "Configuration has errors: " + strings.Join(parts, " | ")
	}

	if len(details.WarningHosts) > 0 || len(details.WarningEndpoints) > 0 {
		var parts []string

		if len(details.WarningHosts) > 0 {
			parts = append(parts, "hosts "+strings.Join(details.WarningHosts, ", "))
		}

		if len(details.WarningEndpoints) > 0 {
			parts = append(parts, "skipped endpoints "+strings.Join(details.WarningEndpoints, ", "))
		}

		return backend.HealthStatusOk, "Configuration valid with warnings: " + strings.Join(parts, " | ")
	}

	return backend.HealthStatusOk, "Successfully connected to all Yamcs hosts and endpoints"
}

func hostDisplayName(hostID string, host *config.YamcsHostConfiguration) string {
	if host.Name != "" {
		return host.Name
	}

	if host.Path != "" {
		return host.Path
	}

	return hostID
}

func endpointDisplayName(endpointID string, endpoint *config.YamcsEndpointConfiguration) string {
	if endpoint.Name != "" {
		return endpoint.Name
	}

	return endpointID
}

func (d *Datasource) GetLastHealthDetails() (json.RawMessage, bool) {
	d.healthMutex.RLock()
	defer d.healthMutex.RUnlock()

	return d.lastHealthDetails, d.lastHealthDetails != nil
}
