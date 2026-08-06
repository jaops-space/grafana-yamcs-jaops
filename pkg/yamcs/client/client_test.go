package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/yamcsManagement"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// mockTransport implements http.RoundTripper to mock HTTP requests.
type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")), // Set body to "{}"
		Header:     make(http.Header),
	}, nil
}

type closeTrackingClientTransport struct {
	closedIdleConnections bool
}

func (m *closeTrackingClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

func (m *closeTrackingClientTransport) CloseIdleConnections() {
	m.closedIdleConnections = true
}

func TestClient(t *testing.T) {

	client, err := NewYamcsClient(
		"somepath",
		corehttp.GetNoTLSConfiguration(),
		&corehttp.NoCredentials{},
		OptionSetProtocol(false),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Assign mockTransport to the client's HTTP transport
	client.HTTP.Client.Transport = &mockTransport{}
	ctx := context.Background()

	instance, err := client.GetInstanceByName(ctx, "someinstance")
	if err != nil {
		t.Fatalf("Failed to get instance: %v", err)
	}

	someProcessor := &yamcsManagement.ProcessorInfo{}
	someProcessor.Name = new("someprocessor")
	instance.Processors = []Processor{someProcessor}

	_, err = client.GetProcessor(instance, "someprocessor")
	if err != nil {
		t.Fatalf("Failed to get processor: %v", err)
	}

	parameter, err := client.GetParameter(ctx, instance.GetName(), "someparameter")
	if err != nil {
		t.Fatalf("Failed to get parameter: %v", err)
	}

	_, err = client.GetParameterValue(ctx, instance, someProcessor, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter value: %v", err)
	}

	_, err = client.GetParameterRanges(ctx, instance, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter ranges: %v", err)
	}

	_, err = client.GetParameterSamples(ctx, instance, parameter, time.Now(), time.Now(), 100)
	if err != nil {
		t.Fatalf("Failed to get parameter samples: %v", err)
	}

	_, err = client.GetCommand(ctx, instance.GetName(), "somecommand")
	if err != nil {
		t.Fatalf("Failed to get command: %v", err)
	}

	_, err = client.IssueCommand(ctx, instance.GetName(), someProcessor.GetName(), "somecommand", make(map[string]any))
	if err != nil {
		t.Fatalf("Failed to issue command: %v", err)
	}

}

func TestClientCloseDisposesHTTPAndClearsSubscriptions(t *testing.T) {
	transport := &closeTrackingClientTransport{}
	client, err := NewYamcsClient(
		"somepath",
		corehttp.GetNoTLSConfiguration(),
		&corehttp.BearerCredentials{
			AccessToken: "access-token",
			Expiry:      time.Now().Add(time.Hour),
		},
		OptionSetProtocol(false),
		OptionSetHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.ParameterSubscriptions[1] = nil
	client.EventSubscriptions[2] = nil
	client.CommandHistorySubscriptions[3] = nil
	client.AlarmSubscriptions[4] = nil
	client.GlobalAlarmStatusSubscriptions[5] = nil
	client.TimeSubscriptions[6] = nil
	client.LinkSubscriptions[7] = nil
	client.ProcessorSubscriptions[8] = nil
	client.HTTP.StartAutoRefresh()

	if client.HTTP.RefreshStop == nil {
		t.Fatal("expected refresh loop to start")
	}

	_ = client.Close()

	if client.HTTP.RefreshStop != nil {
		t.Fatal("expected refresh loop to stop")
	}
	if !transport.closedIdleConnections {
		t.Fatal("expected HTTP idle connections to close")
	}
	if len(client.ParameterSubscriptions) != 0 ||
		len(client.EventSubscriptions) != 0 ||
		len(client.CommandHistorySubscriptions) != 0 ||
		len(client.AlarmSubscriptions) != 0 ||
		len(client.GlobalAlarmStatusSubscriptions) != 0 ||
		len(client.TimeSubscriptions) != 0 ||
		len(client.LinkSubscriptions) != 0 ||
		len(client.ProcessorSubscriptions) != 0 {
		t.Fatal("expected all subscriptions to be cleared on close")
	}
}
