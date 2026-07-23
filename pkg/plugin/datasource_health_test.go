package plugin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
)

func validConfig() *config.YamcsPluginConfiguration {
	return &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"h1": {
				Name: "Primary",
				Path: "localhost:8090",
			},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			"e1": {
				Name:     "Endpoint One",
				Host:     "h1",
				Instance: "sim",
			},
		},
	}
}

func TestValidateConfigItems_BaseErrors(t *testing.T) {
	d := &Datasource{}

	tests := []struct {
		name       string
		cfg        *config.YamcsPluginConfiguration
		wantErrSub string
	}{
		{name: "nil config", cfg: nil, wantErrSub: "configuration is nil"},
		{name: "missing hosts", cfg: &config.YamcsPluginConfiguration{Endpoints: map[string]*config.YamcsEndpointConfiguration{}}, wantErrSub: "hosts configuration is missing"},
		{name: "missing endpoints", cfg: &config.YamcsPluginConfiguration{Hosts: map[string]*config.YamcsHostConfiguration{}}, wantErrSub: "endpoints configuration is missing"},
		{name: "no hosts", cfg: &config.YamcsPluginConfiguration{Hosts: map[string]*config.YamcsHostConfiguration{}, Endpoints: map[string]*config.YamcsEndpointConfiguration{"e1": {Host: "h1", Instance: "sim"}}}, wantErrSub: "no hosts configured"},
		{name: "no endpoints", cfg: &config.YamcsPluginConfiguration{Hosts: map[string]*config.YamcsHostConfiguration{"h1": {Path: "localhost:8090"}}, Endpoints: map[string]*config.YamcsEndpointConfiguration{}}, wantErrSub: "no endpoints configured"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details, hasErrors, err := d.validateConfigItems(tt.cfg)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErrSub, err.Error())
			}
			if !hasErrors {
				t.Fatalf("expected hasValidationErrors=true")
			}
			if details == nil {
				t.Fatalf("expected non-nil details")
			}
		})
	}
}

func TestValidateConfigItems_HostAndEndpointValidationFailures(t *testing.T) {
	d := &Datasource{}
	cfg := &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"badHost": {
				Name: "Broken Host",
				Path: "not-a-host-port",
			},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			"eInvalidHost": {
				Name:     "Endpoint Broken By Host",
				Host:     "badHost",
				Instance: "sim",
			},
			"eUnknownHost": {
				Name:     "Endpoint Unknown Host",
				Host:     "missing",
				Instance: "sim",
			},
		},
	}

	details, hasErrors, err := d.validateConfigItems(cfg)
	if err != nil {
		t.Fatalf("expected no top-level error, got %v", err)
	}
	if !hasErrors {
		t.Fatalf("expected validation errors")
	}

	hostStatus := details.Hosts["badHost"]
	if hostStatus.Status != "error" {
		t.Fatalf("expected badHost status error, got %q", hostStatus.Status)
	}
	if len(details.ErrorHosts) != 1 || details.ErrorHosts[0] != "Broken Host" {
		t.Fatalf("expected ErrorHosts to contain display name, got %#v", details.ErrorHosts)
	}

	if got := details.Endpoints["eInvalidHost"].Message; got != "Host configuration is invalid" {
		t.Fatalf("expected endpoint host invalid message, got %q", got)
	}
	if got := details.Endpoints["eUnknownHost"].Message; got != "References unknown host 'missing'" {
		t.Fatalf("expected unknown host message, got %q", got)
	}
}

func TestApplyConnectivityChecks_NetworkFreeBranches(t *testing.T) {
	d := &Datasource{}
	cfg := &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"h1": {Name: "Host 1", Path: "localhost:8090"},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{
			"missingHostEndpoint": {
				Name:     "No Host",
				Host:     "missing",
				Instance: "sim",
			},
			"warningHostEndpoint": {
				Name:     "Warn Host",
				Host:     "h1",
				Instance: "sim",
			},
		},
	}

	// Force host checks to skip network by marking host non-ok up front.
	details := &HealthDetails{
		Hosts: map[string]ItemStatus{
			"h1": warningStatus("pre-marked unavailable"),
		},
		Endpoints: map[string]ItemStatus{
			"missingHostEndpoint": okStatus(),
			"warningHostEndpoint": okStatus(),
		},
	}

	d.applyConnectivityChecks(context.Background(), cfg, &config.YamcsSecureConfiguration{}, details)

	if got := details.Endpoints["missingHostEndpoint"].Message; got != "Host configuration not found" {
		t.Fatalf("expected missing host message, got %q", got)
	}
	if got := details.Endpoints["warningHostEndpoint"].Status; got != "warning" {
		t.Fatalf("expected warning endpoint status, got %q", got)
	}
	if got := details.Endpoints["warningHostEndpoint"].Message; got != "Skipped because host is unavailable" {
		t.Fatalf("expected skipped warning message, got %q", got)
	}
}

func TestBuildHealthSummary(t *testing.T) {
	t.Run("errors", func(t *testing.T) {
		status, message := buildHealthSummary(&HealthDetails{
			ErrorHosts:     []string{"Host A"},
			ErrorEndpoints: []string{"Endpoint A"},
		})
		if status != backend.HealthStatusError {
			t.Fatalf("expected error status, got %v", status)
		}
		if !strings.Contains(message, "hosts Host A") || !strings.Contains(message, "endpoints Endpoint A") {
			t.Fatalf("unexpected error summary: %q", message)
		}
	})

	t.Run("warnings", func(t *testing.T) {
		status, message := buildHealthSummary(&HealthDetails{
			WarningHosts:     []string{"Host W"},
			WarningEndpoints: []string{"Endpoint W"},
		})
		if status != backend.HealthStatusOk {
			t.Fatalf("expected ok status for warnings, got %v", status)
		}
		if !strings.Contains(message, "hosts Host W") || !strings.Contains(message, "skipped endpoints Endpoint W") {
			t.Fatalf("unexpected warning summary: %q", message)
		}
	})

	t.Run("success", func(t *testing.T) {
		status, message := buildHealthSummary(&HealthDetails{})
		if status != backend.HealthStatusOk {
			t.Fatalf("expected ok status, got %v", status)
		}
		if message != "Successfully connected to all Yamcs hosts and endpoints" {
			t.Fatalf("unexpected success summary: %q", message)
		}
	})
}

func TestCheckHealth_StoresLastDetailsOnValidationError(t *testing.T) {
	d := &Datasource{}
	settingsCfg := validConfig()
	settingsCfg.Hosts = map[string]*config.YamcsHostConfiguration{}

	jsonData, err := json.Marshal(settingsCfg)
	if err != nil {
		t.Fatalf("failed to marshal settings config: %v", err)
	}

	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
				JSONData: jsonData,
			},
		},
	}

	result, err := d.CheckHealth(context.Background(), req)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if result.Status != backend.HealthStatusError {
		t.Fatalf("expected error status, got %v", result.Status)
	}
	if result.Message != "no hosts configured" {
		t.Fatalf("expected validation message, got %q", result.Message)
	}

	_, ok := d.GetLastHealthDetails()
	if !ok {
		t.Fatalf("expected stored health details")
	}
}
