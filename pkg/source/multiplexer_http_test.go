package source

import (
	"context"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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

func TestMultiplexerUsesDatasourceTLSOptionsForWebSocket(t *testing.T) {
	tests := []struct {
		name        string
		tlsInsecure bool
		useCustomCA bool
	}{
		{name: "custom CA", useCustomCA: true},
		{name: "skip verification", tlsInsecure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, req, nil)
				if err != nil {
					t.Errorf("upgrade WebSocket: %v", err)
					return
				}
				defer conn.Close()
				_, _, _ = conn.ReadMessage()
			}))
			defer server.Close()

			opts := httpclient.Options{}
			if tt.useCustomCA {
				opts.TLS = &httpclient.TLSOptions{
					CACertificate: string(pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: server.Certificate().Raw,
					})),
				}
			}

			cfg := &config.YamcsPluginConfiguration{
				Hosts: map[string]*config.YamcsHostConfiguration{
					"host": {
						Path:        strings.TrimPrefix(server.URL, "https://"),
						Tls:         true,
						TlsInsecure: tt.tlsInsecure,
					},
				},
				Endpoints: map[string]*config.YamcsEndpointConfiguration{},
			}
			mux, err := NewMultiplexerWithContext(context.Background(), cfg, &config.YamcsSecureConfiguration{}, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer mux.Dispose()

			ws := mux.Hosts["host"].Client.WebSocket
			if err := ws.Connect(context.Background()); err != nil {
				t.Fatalf("connect WebSocket with datasource TLS options: %v", err)
			}
			defer ws.Disconnect()
		})
	}
}
