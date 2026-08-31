package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/alarms"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
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

func TestParameterListenerBuffersExpiredValues(t *testing.T) {
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

	if got := endpoint.GetAndClearParameterStreamBuffer("/SIM/TEMP", "req/sim/temp"); len(got) != 1 {
		t.Fatalf("expected expired value to be buffered, got %d values", len(got))
	}
}

func TestParameterListenerIgnoresInvalidValues(t *testing.T) {
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
		AcquisitionStatus: pvalue.AcquisitionStatus_INVALID.Enum(),
	}); err != nil {
		t.Fatalf("listener returned error: %v", err)
	}

	if got := endpoint.GetAndClearParameterStreamBuffer("/SIM/TEMP", "req/sim/temp"); len(got) != 0 {
		t.Fatalf("expected invalid value to be ignored, got %d buffered values", len(got))
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

func TestSetUnitAndThresholdsOnlyConfiguresParameterField(t *testing.T) {
	okThreshold := data.NewThreshold(0, "green", "")
	warnThreshold := data.NewThreshold(50, "red", "")
	endpoint := &YamcsEndpoint{
		Parameters: map[string]*ParameterDemand{
			"/SIM/TEMP": {
				Name: "/SIM/TEMP",
				Unit: "degC",
				Thresholds: []*data.Threshold{
					&okThreshold,
					&warnThreshold,
				},
			},
		},
	}
	frame := data.NewFrame("response",
		data.NewField("time", nil, []time.Time{time.Unix(0, 0)}),
		data.NewField("/SIM/TEMP", nil, []float64{42}),
		data.NewField("min(/SIM/TEMP)", nil, []float64{40}),
		data.NewField("max(/SIM/TEMP)", nil, []float64{44}),
	)

	endpoint.SetUnitAndThresholds(context.Background(), "/SIM/TEMP", frame)

	if frame.Fields[0].Config != nil {
		t.Fatalf("expected time field config to stay nil, got %#v", frame.Fields[0].Config)
	}
	if frame.Fields[2].Config != nil {
		t.Fatalf("expected min field config to stay nil, got %#v", frame.Fields[2].Config)
	}
	if frame.Fields[3].Config != nil {
		t.Fatalf("expected max field config to stay nil, got %#v", frame.Fields[3].Config)
	}

	valueConfig := frame.Fields[1].Config
	if valueConfig == nil {
		t.Fatalf("expected parameter field config")
	}
	if valueConfig.Unit != "degC" {
		t.Fatalf("expected unit degC, got %q", valueConfig.Unit)
	}
	if valueConfig.Thresholds == nil {
		t.Fatalf("expected thresholds config")
	}
	if got := len(valueConfig.Thresholds.Steps); got != 2 {
		t.Fatalf("expected 2 thresholds, got %d", got)
	}
}

// slowJSONTransport is an http.RoundTripper that sleeps for delay before
// returning a successful, empty JSON body - simulating a slow-but-working
// HTTP call (e.g. a loaded/high-latency Yamcs instance), as opposed to an
// outright failure.
type slowJSONTransport struct {
	delay time.Duration
}

func (t *slowJSONTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(t.delay)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// TestRequestNewParameterStreamDoesNotBlockUnrelatedEndpointState verifies
// that a slow parameter-demand creation (e.g. a slow GetParameter HTTP call
// for a brand new parameter, encountered via the real RequestNewParameterStream
// entry point) does not hold mu, so unrelated endpoint bookkeeping (here:
// alarms) for the very same endpoint stays fully responsive while it's in
// flight. Previously mu was held for RequestNewParameterStream's entire body
// (including this HTTP call), so one slow/stuck subscribe-ish attempt for
// one data type could stall every other panel on the same endpoint.
func TestRequestNewParameterStreamDoesNotBlockUnrelatedEndpointState(t *testing.T) {
	cli, err := client.NewYamcsClient("unused", corehttp.GetNoTLSConfiguration(), &corehttp.NoCredentials{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	cli.HTTP.Client.Transport = &slowJSONTransport{delay: 300 * time.Millisecond}

	endpoint := &YamcsEndpoint{
		Host:          &YamcsHost{Client: cli, Configuration: &config.YamcsHostConfiguration{ID: "test-host"}},
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters:    map[string]*ParameterDemand{},
		Alarms:        map[string][]*alarms.AlarmData{},
		AlarmSignals:  map[string]chan struct{}{},
	}

	requestStarted := make(chan struct{})
	requestDone := make(chan struct{})
	go func() {
		close(requestStarted)
		// The instance/processor aren't wired up on the host, so this will
		// error out after creating the demand - that's fine, we only care
		// about how long mu is held while it's in flight.
		_ = endpoint.RequestNewParameterStream(context.Background(), "/SIM/NEW_PARAM", "some/req/path")
		close(requestDone)
	}()

	<-requestStarted
	time.Sleep(50 * time.Millisecond) // give the goroutine time to be mid-HTTP-call

	start := time.Now()
	endpoint.ClearAlarmsStream("some/path")
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("unrelated endpoint state (alarms) took %v to update while a parameter stream was being requested - mu is being held across the slow call", elapsed)
	}

	select {
	case <-requestDone:
	case <-time.After(2 * time.Second):
		t.Fatal("RequestNewParameterStream never completed")
	}
}

// TestGetOrCreateParameterDemandDoesNotBlockUnrelatedEndpointState verifies
// that a slow parameter-demand creation (e.g. a slow GetParameter HTTP call
// for a brand new parameter) does not hold mu, so unrelated endpoint
// bookkeeping (here: alarms) for the very same endpoint stays fully
// responsive while it's in flight. Previously mu was held across this call,
// so one slow/stuck subscribe-ish attempt for one data type could stall
// every other panel on the same endpoint.
func TestGetOrCreateParameterDemandDoesNotBlockUnrelatedEndpointState(t *testing.T) {
	cli, err := client.NewYamcsClient("unused", corehttp.GetNoTLSConfiguration(), &corehttp.NoCredentials{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	cli.HTTP.Client.Transport = &slowJSONTransport{delay: 300 * time.Millisecond}

	endpoint := &YamcsEndpoint{
		Host:          &YamcsHost{Client: cli, Configuration: &config.YamcsHostConfiguration{ID: "test-host"}},
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters:    map[string]*ParameterDemand{},
		Alarms:        map[string][]*alarms.AlarmData{},
		AlarmSignals:  map[string]chan struct{}{},
	}

	demandStarted := make(chan struct{})
	demandDone := make(chan struct{})
	go func() {
		close(demandStarted)
		_, _ = endpoint.getOrCreateParameterDemand(context.Background(), "/SIM/NEW_PARAM")
		close(demandDone)
	}()

	<-demandStarted
	time.Sleep(50 * time.Millisecond) // give the goroutine time to be mid-HTTP-call

	start := time.Now()
	endpoint.ClearAlarmsStream("some/path")
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("unrelated endpoint state (alarms) took %v to update while a parameter demand was being created - mu is being held across the slow call", elapsed)
	}

	select {
	case <-demandDone:
	case <-time.After(2 * time.Second):
		t.Fatal("getOrCreateParameterDemand never completed")
	}
}

// TestParameterListenerDropsUnknownParameterWithoutNetworkCall verifies that
// getChannelParameterListener - which runs synchronously on the WebSocket's
// single read loop - never attempts a live (HTTP) demand lookup for a
// parameter it doesn't already know about. Doing so would block delivery of
// every other incoming message on the same connection for as long as that
// call takes. Instead, values for unknown parameters must simply be dropped.
func TestParameterListenerDropsUnknownParameterWithoutNetworkCall(t *testing.T) {
	cli, err := client.NewYamcsClient("unused", corehttp.GetNoTLSConfiguration(), &corehttp.NoCredentials{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	// If the listener ever attempted a network call, this transport would
	// make the test hang/timeout, catching the regression.
	cli.HTTP.Client.Transport = &slowJSONTransport{delay: 5 * time.Second}

	endpoint := &YamcsEndpoint{
		Host:          &YamcsHost{Client: cli, Configuration: &config.YamcsHostConfiguration{ID: "test-host"}},
		Configuration: &config.YamcsEndpointConfiguration{Instance: "sim", Processor: "realtime"},
		Parameters:    map[string]*ParameterDemand{},
	}

	listener := endpoint.getChannelParameterListener()

	done := make(chan error, 1)
	go func() {
		done <- listener("/SIM/UNKNOWN", &pvalue.ParameterValue{
			AcquisitionStatus: pvalue.AcquisitionStatus_ACQUIRED.Enum(),
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected listener to drop the value without error, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("listener blocked on a network call for an unknown parameter")
	}

	if _, found := endpoint.Parameters["/SIM/UNKNOWN"]; found {
		t.Fatal("expected listener not to create a demand for an unknown parameter")
	}
}
