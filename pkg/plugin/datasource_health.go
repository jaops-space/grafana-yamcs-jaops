package plugin

import (
	"context"
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
			details.ErrorHosts = append(details.ErrorHosts, host.DisplayName())
			hasValidationErrors = true
			continue
		}
	}

	for endpointID, endpoint := range y.Endpoints {
		details.Endpoints[endpointID] = okStatus()

		if err := endpoint.Validate(y); err != nil {
			details.Endpoints[endpointID] = errorStatus(err.Error())
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			continue
		}

		if endpoint.Host == "" {
			details.Endpoints[endpointID] = errorStatus("No host selected")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			continue
		}

		if _, exists := y.Hosts[endpoint.Host]; !exists {
			details.Endpoints[endpointID] = errorStatus(fmt.Sprintf("References unknown host '%s'", endpoint.Host))
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			continue
		}

		if hostStatus, exists := details.Hosts[endpoint.Host]; exists && hostStatus.Status == "error" {
			details.Endpoints[endpointID] = errorStatus("Host configuration is invalid")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			continue
		}
	}

	return details, hasValidationErrors, nil
}

func (d *Datasource) applyConnectivityChecks(
	ctx context.Context,
	cfg *config.YamcsPluginConfiguration,
	seccfg *config.YamcsSecureConfiguration,
	details *HealthDetails,
) error {
	connectivityCfg := &config.YamcsPluginConfiguration{
		Hosts:     map[string]*config.YamcsHostConfiguration{},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{},
	}

	for hostID, host := range cfg.Hosts {
		connectivityCfg.Hosts[hostID] = host
	}

	for epID, ep := range cfg.Endpoints {
		hostStatus, hostConfigured := details.Hosts[ep.Host]
		if !hostConfigured {
			details.Endpoints[epID] = errorStatus("Host configuration not found")
			details.ErrorEndpoints = append(details.ErrorEndpoints, ep.DisplayName())
			continue
		}

		if hostStatus.Status != "ok" {
			details.Endpoints[epID] = warningStatus("Skipped because host is unavailable")
			details.WarningEndpoints = append(details.WarningEndpoints, ep.DisplayName())
			continue
		}

		connectivityCfg.Endpoints[epID] = ep
	}

	if len(connectivityCfg.Endpoints) == 0 {
		return nil
	}

	testMux, err := source.NewMultiplexer(connectivityCfg, seccfg)
	if err != nil {
		return err
	}

	hostErrors, epsErrors := testMux.Connect(ctx, false)

	for hostID, host := range cfg.Hosts {
		if current, exists := details.Hosts[hostID]; exists && current.Status != "ok" {
			continue
		}

		if err, hasError := hostErrors[hostID]; hasError {
			details.Hosts[hostID] = errorStatus(err.Error())
			details.ErrorHosts = append(details.ErrorHosts, host.DisplayName())
			continue
		}
		details.Hosts[hostID] = okStatus()
	}

	for epID, ep := range connectivityCfg.Endpoints {
		if _, hostHasError := hostErrors[ep.Host]; hostHasError {
			details.Endpoints[epID] = warningStatus("skipped because host has an error")
			details.WarningEndpoints = append(details.WarningEndpoints, ep.DisplayName())
			continue
		}
		if err, hasError := epsErrors[epID]; hasError {
			details.Endpoints[epID] = errorStatus(err.Error())
			details.ErrorEndpoints = append(details.ErrorEndpoints, ep.DisplayName())
			continue
		}
		details.Endpoints[epID] = okStatus()
	}

	testMux.Dispose()

	return nil
}

func (d *Datasource) refreshHealthDetails(ctx context.Context) (*HealthDetails, error) {
	if d.multiplexer == nil || d.multiplexer.Config == nil {
		return nil, exception.New("could not find multiplexer configuration", "REFRESH_HEALTH_NO_CONFIG")
	}

	secure := d.multiplexer.Secure
	if secure == nil {
		secure = &config.YamcsSecureConfiguration{Hosts: map[string]*config.YamcsSecureHost{}}
	}

	details, hasValidationErrors, err := d.validateConfigItems(d.multiplexer.Config)
	if err != nil {
		return nil, err
	}

	if !hasValidationErrors {
		if err := d.applyConnectivityChecks(ctx, d.multiplexer.Config, secure, details); err != nil {
			return nil, err
		}
	}

	d.healthMutex.Lock()
	d.lastHealthDetails = details
	d.healthMutex.Unlock()

	return details, nil
}

func (d *Datasource) evaluateHealth(
	ctx context.Context,
	y *config.YamcsPluginConfiguration,
	secure *config.YamcsSecureConfiguration,
) (backend.HealthStatus, string, *HealthDetails, error) {
	details, hasValidationErrors, err := d.validateConfigItems(y)
	if err != nil {
		return backend.HealthStatusError, err.Error(), details, nil
	}

	if !hasValidationErrors {
		err = d.applyConnectivityChecks(ctx, y, secure, details)
		if err != nil {
			return backend.HealthStatusError, "health connectivity check failed", details, err
		}
	}

	status, message := buildHealthSummary(details)

	return status, message, details, nil
}

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings := req.PluginContext.DataSourceInstanceSettings
	if settings == nil {
		return nil, exception.New("Datasource instance settings are nil", "HEALTH_SETTINGS_NIL")
	}

	cfg, secure, err := config.ExtractConfig(*settings)
	if err != nil {
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	status, message, details, err := d.evaluateHealth(ctx, cfg, secure)
	if err != nil {
		return nil, err
	}

	d.healthMutex.Lock()
	d.lastHealthDetails = details
	d.healthMutex.Unlock()

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

func healthStatusCounts(details *HealthDetails) (hostOK, hostWarn, hostErr, endpointOK, endpointWarn, endpointErr int) {
	for _, status := range details.Hosts {
		switch status.Status {
		case "ok":
			hostOK++
		case "warning":
			hostWarn++
		case "error":
			hostErr++
		}
	}

	for _, status := range details.Endpoints {
		switch status.Status {
		case "ok":
			endpointOK++
		case "warning":
			endpointWarn++
		case "error":
			endpointErr++
		}
	}

	return hostOK, hostWarn, hostErr, endpointOK, endpointWarn, endpointErr
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

func (d *Datasource) GetLastHealthDetails() (*HealthDetails, bool) {
	d.healthMutex.RLock()
	defer d.healthMutex.RUnlock()

	return d.lastHealthDetails, d.lastHealthDetails != nil
}
