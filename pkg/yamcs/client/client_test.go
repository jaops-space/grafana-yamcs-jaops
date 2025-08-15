package client

import (
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

	instance, err := client.GetInstanceByName("someinstance")
	if err != nil {
		t.Fatalf("Failed to get instance: %v", err)
	}

	someProcessor := &yamcsManagement.ProcessorInfo{}
	someProcessor.Name = new(string)
	*someProcessor.Name = "someprocessor"
	instance.Processors = []Processor{someProcessor}

	_, err = client.GetProcessor(instance, "someprocessor")
	if err != nil {
		t.Fatalf("Failed to get processor: %v", err)
	}

	parameter, err := client.GetParameter(instance, "someparameter")
	if err != nil {
		t.Fatalf("Failed to get parameter: %v", err)
	}

	_, err = client.GetParameterValue(instance, someProcessor, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter value: %v", err)
	}

	_, err = client.GetParameterRanges(instance, parameter)
	if err != nil {
		t.Fatalf("Failed to get parameter ranges: %v", err)
	}

	_, err = client.GetParameterSamples(instance, parameter, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to get parameter samples: %v", err)
	}

	_, err = client.GetCommand(instance.GetName(), "somecommand")
	if err != nil {
		t.Fatalf("Failed to get command: %v", err)
	}

	_, err = client.IssueCommand(instance, someProcessor, "somecommand", make(map[string]any))
	if err != nil {
		t.Fatalf("Failed to issue command: %v", err)
	}

}
