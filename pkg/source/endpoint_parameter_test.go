package source

import (
	"context"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
)

func TestParameterListenerBuffersOncePerUniqueStreamDemand(t *testing.T) {
	endpoint := &YamcsEndpoint{
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters: map[string]*ParameterDemand{
			"/SIM/TEMP": {
				Name: "/SIM/TEMP",
				Streams: map[string]*ParameterStreamDemand{
					"req/sim/temp": {Path: "req/sim/temp", Buffer: []*pvalue.ParameterValue{}},
				},
			},
		},
	}
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp"].parameter = endpoint.Parameters["/SIM/TEMP"]

	listener := endpoint.getChannelParameterListener()
	if err := listener("/SIM/TEMP", &pvalue.ParameterValue{
		AcquisitionStatus: pvalue.AcquisitionStatus_ACQUIRED.Enum(),
	}); err != nil {
		t.Fatalf("listener returned error: %v", err)
	}

	buffer := endpoint.GetAndClearParameterStreamBuffer("/SIM/TEMP", "req/sim/temp")
	if len(buffer) != 1 {
		t.Fatalf("expected exactly one buffered value for unique stream demand, got %d", len(buffer))
	}
	if got := endpoint.GetAndClearParameterStreamBuffer("/SIM/TEMP", "req/sim/temp"); len(got) != 0 {
		t.Fatalf("expected buffer to be cleared, got %d values", len(got))
	}
}

func TestParameterListenerIgnoresNonAcquiredValues(t *testing.T) {
	endpoint := &YamcsEndpoint{
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters: map[string]*ParameterDemand{
			"/SIM/TEMP": {
				Name: "/SIM/TEMP",
				Streams: map[string]*ParameterStreamDemand{
					"req/sim/temp": {Path: "req/sim/temp", Buffer: []*pvalue.ParameterValue{}},
				},
			},
		},
	}
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp"].parameter = endpoint.Parameters["/SIM/TEMP"]

	listener := endpoint.getChannelParameterListener()
	if err := listener("/SIM/TEMP", &pvalue.ParameterValue{
		AcquisitionStatus: pvalue.AcquisitionStatus_EXPIRED.Enum(),
	}); err != nil {
		t.Fatalf("listener returned error: %v", err)
	}

	if got := endpoint.GetAndClearParameterStreamBuffer("/SIM/TEMP", "req/sim/temp"); len(got) != 0 {
		t.Fatalf("expected non-acquired value to be ignored, got %d buffered values", len(got))
	}
}

func TestParameterListenerProcessObserverReportsStreamCount(t *testing.T) {
	endpoint := &YamcsEndpoint{
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters: map[string]*ParameterDemand{
			"/SIM/TEMP": {
				Name: "/SIM/TEMP",
				Streams: map[string]*ParameterStreamDemand{
					"req/sim/temp/1": {Path: "req/sim/temp/1", Buffer: []*pvalue.ParameterValue{}},
					"req/sim/temp/2": {Path: "req/sim/temp/2", Buffer: []*pvalue.ParameterValue{}},
				},
			},
		},
	}
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp/1"].parameter = endpoint.Parameters["/SIM/TEMP"]
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp/2"].parameter = endpoint.Parameters["/SIM/TEMP"]

	calls := 0
	observedStreamCount := 0
	endpoint.ParameterProcessObserver = func(parameter string, streamCount int, _ time.Duration) {
		calls++
		if parameter != "/SIM/TEMP" {
			t.Fatalf("expected observer parameter /SIM/TEMP, got %s", parameter)
		}
		observedStreamCount = streamCount
	}

	listener := endpoint.getChannelParameterListener()
	if err := listener("/SIM/TEMP", &pvalue.ParameterValue{
		AcquisitionStatus: pvalue.AcquisitionStatus_ACQUIRED.Enum(),
	}); err != nil {
		t.Fatalf("listener returned error: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected observer to be called once, got %d", calls)
	}
	if observedStreamCount != 2 {
		t.Fatalf("expected observer stream count 2, got %d", observedStreamCount)
	}
}

func TestParameterListenerBufferObserverReportsStreamPaths(t *testing.T) {
	endpoint := &YamcsEndpoint{
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters: map[string]*ParameterDemand{
			"/SIM/TEMP": {
				Name: "/SIM/TEMP",
				Streams: map[string]*ParameterStreamDemand{
					"req/sim/temp/1": {Path: "req/sim/temp/1", Buffer: []*pvalue.ParameterValue{}},
					"req/sim/temp/2": {Path: "req/sim/temp/2", Buffer: []*pvalue.ParameterValue{}},
				},
			},
		},
	}
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp/1"].parameter = endpoint.Parameters["/SIM/TEMP"]
	endpoint.Parameters["/SIM/TEMP"].Streams["req/sim/temp/2"].parameter = endpoint.Parameters["/SIM/TEMP"]

	observed := map[string]bool{}
	endpoint.ParameterBufferObserver = func(parameter string, path string, receivedAt time.Time) {
		if parameter != "/SIM/TEMP" {
			t.Fatalf("expected observer parameter /SIM/TEMP, got %s", parameter)
		}
		if receivedAt.IsZero() {
			t.Fatalf("expected non-zero receivedAt")
		}
		observed[path] = true
	}

	listener := endpoint.getChannelParameterListener()
	if err := listener("/SIM/TEMP", &pvalue.ParameterValue{
		AcquisitionStatus: pvalue.AcquisitionStatus_ACQUIRED.Enum(),
	}); err != nil {
		t.Fatalf("listener returned error: %v", err)
	}

	for _, path := range []string{"req/sim/temp/1", "req/sim/temp/2"} {
		if !observed[path] {
			t.Fatalf("expected observer to report path %s", path)
		}
	}
}

func TestYamcsHostIsConnectedWithNilClient(t *testing.T) {
	host := &YamcsHost{}

	if host.IsConnected() {
		t.Fatalf("expected nil-client host to be disconnected")
	}
}

func TestWithdrawUnknownParameterStreamIsNoop(t *testing.T) {
	endpoint := &YamcsEndpoint{Parameters: map[string]*ParameterDemand{}}

	if err := endpoint.WithdrawParameterStreamRequest(context.Background(), "/SIM/TEMP", "req/sim/temp"); err != nil {
		t.Fatalf("expected withdrawing unknown stream to be a no-op, got %v", err)
	}
}
