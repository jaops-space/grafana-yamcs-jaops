package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	start := time.Now()
	backend.Logger.Debug("health.validate.start")
	defer func() {
		backend.Logger.Debug("health.validate.finish", "durationMs", time.Since(start).Milliseconds())
	}()

	details := &HealthDetails{
		Hosts:     map[string]ItemStatus{},
		Endpoints: map[string]ItemStatus{},
	}

	if y == nil {
		backend.Logger.Debug("health.validate.nil_config")
		return details, true, fmt.Errorf("configuration is nil")
	}

	details.TotalHosts = len(y.Hosts)
	details.TotalEndpoints = len(y.Endpoints)
	backend.Logger.Debug("health.validate.config_counts", "hosts", details.TotalHosts, "endpoints", details.TotalEndpoints)

	hasValidationErrors := false

	if y.Hosts == nil {
		backend.Logger.Debug("health.validate.hosts_missing")
		return details, true, fmt.Errorf("hosts configuration is missing")
	}

	if y.Endpoints == nil {
		backend.Logger.Debug("health.validate.endpoints_missing")
		return details, true, fmt.Errorf("endpoints configuration is missing")
	}

	if len(y.Hosts) == 0 {
		backend.Logger.Debug("health.validate.hosts_empty")
		return details, true, fmt.Errorf("no hosts configured")
	}

	if len(y.Endpoints) == 0 {
		backend.Logger.Debug("health.validate.endpoints_empty")
		return details, true, fmt.Errorf("no endpoints configured")
	}

	for hostID, host := range y.Hosts {
		hostStart := time.Now()
		details.Hosts[hostID] = okStatus()
		backend.Logger.Debug("health.validate.host.start", "hostID", hostID, "displayName", host.DisplayName(), "path", host.Path, "authEnabled", host.Auth, "tlsEnabled", host.Tls)

		if err := host.Validate(y); err != nil {
			details.Hosts[hostID] = errorStatus(err.Error())
			details.ErrorHosts = append(details.ErrorHosts, host.DisplayName())
			hasValidationErrors = true
			continue
		}

		backend.Logger.Debug("health.validate.host.ok", "hostID", hostID, "durationMs", time.Since(hostStart).Milliseconds())
	}

	for endpointID, endpoint := range y.Endpoints {
		endpointStart := time.Now()
		details.Endpoints[endpointID] = okStatus()
		backend.Logger.Debug("health.validate.endpoint.start", "endpointID", endpointID, "displayName", endpoint.DisplayName(), "host", endpoint.Host, "instance", endpoint.Instance, "processor", endpoint.Processor)

		if err := endpoint.Validate(y); err != nil {
			details.Endpoints[endpointID] = errorStatus(err.Error())
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			backend.Logger.Debug("health.validate.endpoint.error", "endpointID", endpointID, "error", err.Error(), "durationMs", time.Since(endpointStart).Milliseconds())
			continue
		}

		if endpoint.Host == "" {
			details.Endpoints[endpointID] = errorStatus("No host selected")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			backend.Logger.Debug("health.validate.endpoint.error", "endpointID", endpointID, "error", "No host selected", "durationMs", time.Since(endpointStart).Milliseconds())
			continue
		}

		if _, exists := y.Hosts[endpoint.Host]; !exists {
			details.Endpoints[endpointID] = errorStatus(fmt.Sprintf("References unknown host '%s'", endpoint.Host))
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			backend.Logger.Debug("health.validate.endpoint.error", "endpointID", endpointID, "error", fmt.Sprintf("References unknown host '%s'", endpoint.Host), "durationMs", time.Since(endpointStart).Milliseconds())
			continue
		}

		if hostStatus, exists := details.Hosts[endpoint.Host]; exists && hostStatus.Status == "error" {
			details.Endpoints[endpointID] = errorStatus("Host configuration is invalid")
			details.ErrorEndpoints = append(details.ErrorEndpoints, endpoint.DisplayName())
			hasValidationErrors = true
			backend.Logger.Debug("health.validate.endpoint.error", "endpointID", endpointID, "error", "Host configuration is invalid", "durationMs", time.Since(endpointStart).Milliseconds())
			continue
		}

		backend.Logger.Debug("health.validate.endpoint.ok", "endpointID", endpointID, "durationMs", time.Since(endpointStart).Milliseconds())
	}

	hostOK, hostWarn, hostErr, endpointOK, endpointWarn, endpointErr := healthStatusCounts(details)
	backend.Logger.Debug("health.validate.summary",
		"hasValidationErrors", hasValidationErrors,
		"hostOK", hostOK,
		"hostWarn", hostWarn,
		"hostErr", hostErr,
		"endpointOK", endpointOK,
		"endpointWarn", endpointWarn,
		"endpointErr", endpointErr,
	)

	return details, hasValidationErrors, nil
}

func (d *Datasource) applyConnectivityChecks(
	ctx context.Context,
	cfg *config.YamcsPluginConfiguration,
	seccfg *config.YamcsSecureConfiguration,
	details *HealthDetails,
) error {
	start := time.Now()
	backend.Logger.Debug("health.connectivity.start", "hostCount", len(cfg.Hosts), "endpointCount", len(cfg.Endpoints))
	defer func() {
		hostOK, hostWarn, hostErr, endpointOK, endpointWarn, endpointErr := healthStatusCounts(details)
		backend.Logger.Debug("health.connectivity.finish",
			"durationMs", time.Since(start).Milliseconds(),
			"hostOK", hostOK,
			"hostWarn", hostWarn,
			"hostErr", hostErr,
			"endpointOK", endpointOK,
			"endpointWarn", endpointWarn,
			"endpointErr", endpointErr,
		)
	}()

	testMux, err := source.NewMultiplexer(cfg, seccfg)
	if err != nil {
		return err
	}
	backend.Logger.Debug("health.connectivity.hosts.begin")

	hostErrors, epsErrors := testMux.Connect(ctx, false)

	for hostID, host := range cfg.Hosts {
		if err, hasError := hostErrors[hostID]; hasError {
			details.Hosts[hostID] = errorStatus(err.Error())
			details.ErrorHosts = append(details.ErrorHosts, host.DisplayName())
			continue
		}
		details.Hosts[hostID] = okStatus()
	}

	for epID, ep := range cfg.Endpoints {
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

func (d *Datasource) evaluateHealth(
	ctx context.Context,
	y *config.YamcsPluginConfiguration,
	secure *config.YamcsSecureConfiguration,
) (backend.HealthStatus, string, *HealthDetails, error) {
	start := time.Now()
	backend.Logger.Debug("health.evaluate.start")
	details, hasValidationErrors, err := d.validateConfigItems(y)
	if err != nil {
		backend.Logger.Debug("health.evaluate.validation_top_level_error", "error", err.Error())

		backend.Logger.Debug("health.evaluate.finish", "status", backend.HealthStatusError, "durationMs", time.Since(start).Milliseconds())
		return backend.HealthStatusError, err.Error(), details, nil
	}

	backend.Logger.Debug("health.evaluate.validation_complete", "hasValidationErrors", hasValidationErrors)
	if !hasValidationErrors {
		d.applyConnectivityChecks(ctx, y, secure, details)
	} else {
		backend.Logger.Debug("health.evaluate.connectivity_skipped", "reason", "validation errors found")
	}

	status, message := buildHealthSummary(details)
	hostOK, hostWarn, hostErr, endpointOK, endpointWarn, endpointErr := healthStatusCounts(details)
	backend.Logger.Debug("health.evaluate.finish",
		"status", status,
		"message", message,
		"durationMs", time.Since(start).Milliseconds(),
		"hostOK", hostOK,
		"hostWarn", hostWarn,
		"hostErr", hostErr,
		"endpointOK", endpointOK,
		"endpointWarn", endpointWarn,
		"endpointErr", endpointErr,
	)

	return status, message, details, nil
}

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	start := time.Now()
	backend.Logger.Debug("health.check.start")
	if deadline, ok := ctx.Deadline(); ok {
		backend.Logger.Debug("health.check.deadline", "deadline", deadline.UTC().Format(time.RFC3339Nano), "remainingMs", time.Until(deadline).Milliseconds())
	} else {
		backend.Logger.Debug("health.check.deadline", "deadline", "none")
	}

	settings := req.PluginContext.DataSourceInstanceSettings
	if settings == nil {
		backend.Logger.Debug("health.check.error", "error", "datasource settings are nil")
		return nil, exception.New("Datasource instance settings are nil", "HEALTH_SETTINGS_NIL")
	}

	cfg, secure, err := config.ExtractConfig(*settings)
	if err != nil {
		backend.Logger.Debug("health.check.extract_config_error", "error", err.Error(), "durationMs", time.Since(start).Milliseconds())
		return nil, exception.Wrap("Error loading plugin configuration", "CONFIGURATION_LOAD_ERROR", err)
	}

	backend.Logger.Debug("health.check.config_loaded", "hosts", len(cfg.Hosts), "endpoints", len(cfg.Endpoints))

	status, message, details, err := d.evaluateHealth(ctx, cfg, secure)
	if err != nil {
		backend.Logger.Debug("health.check.evaluate_error", "error", err.Error(), "durationMs", time.Since(start).Milliseconds())
		return nil, err
	}

	d.healthMutex.Lock()
	d.lastHealthDetails = details
	d.healthMutex.Unlock()
	backend.Logger.Debug("health.check.details_saved")

	// save details and return only status and message
	backend.Logger.Debug("health.check.finish", "status", status, "message", message, "durationMs", time.Since(start).Milliseconds())
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
