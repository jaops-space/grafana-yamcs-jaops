package plugin

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
	"google.golang.org/protobuf/encoding/protojson"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDatasourceSingleValueFrameAddsExpiredNotice(t *testing.T) {
	parameter := "/SIM/TEMP"
	responseBody, err := protojson.Marshal(&pvalue.ParameterValue{
		AcquisitionStatus: pvalue.AcquisitionStatus_EXPIRED.Enum(),
		EngValue:          &protobuf.Value{Type: protobuf.Value_FLOAT.Enum(), FloatValue: new(float32(12.5))},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/processors/sim/realtime/parameters/"+parameter {
				t.Fatalf("unexpected Yamcs request path: %s", req.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
				Request:    req,
			}, nil
		}),
	}

	yamcsClient, err := client.NewYamcsClient(
		"yamcs.invalid",
		corehttp.TLS{},
		nil,
		client.OptionSetProtocol(false),
		client.OptionSetHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("create Yamcs client: %v", err)
	}

	endpoint := &source.YamcsEndpoint{
		Host: &source.YamcsHost{Client: yamcsClient},
		Configuration: &config.YamcsEndpointConfiguration{
			Instance:  "sim",
			Processor: "realtime",
		},
		Parameters: map[string]*source.ParameterDemand{
			parameter: {
				Name:       parameter,
				Streams:    map[string]*source.ParameterStreamDemand{},
				Thresholds: []*data.Threshold{},
			},
		},
	}

	frame, err := DatasourceSingleValueFrame(context.Background(), endpoint, PluginQuery{
		Type:      SingleValue,
		Parameter: parameter,
	})
	if err != nil {
		t.Fatalf("single value frame: %v", err)
	}

	if frame.Meta == nil || len(frame.Meta.Notices) != 1 {
		t.Fatalf("expected one expired notice, got %#v", frame.Meta)
	}
	if frame.Meta.Notices[0].Severity != data.NoticeSeverityWarning {
		t.Fatalf("expected warning notice, got %v", frame.Meta.Notices[0].Severity)
	}
	if !strings.Contains(frame.Meta.Notices[0].Text, "expired") {
		t.Fatalf("expected expired notice text, got %q", frame.Meta.Notices[0].Text)
	}
	valueField, _ := frame.FieldByName(parameter)
	if valueField == nil {
		t.Fatalf("expected parameter field %s", parameter)
	}
	if got := valueField.At(0); got != float64(12.5) {
		t.Fatalf("expected expired parameter sample to retain its value, got %v", got)
	}
}
