package source

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
)

func TestMultiplexerUsesDatasourceHTTPOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Configured-Header") != "present" {
			http.Error(w, "missing datasource header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"host": {Path: strings.TrimPrefix(server.URL, "http://")},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{},
	}
	opts := httpclient.Options{
		Header: http.Header{"X-Configured-Header": []string{"present"}},
	}
	mux, err := NewMultiplexerWithContext(context.Background(), cfg, &config.YamcsSecureConfiguration{}, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Dispose()

	resp, err := mux.Hosts["host"].Client.HTTP.Client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("datasource HTTP header not forwarded: status %d, body %s", resp.StatusCode, body)
	}
}
