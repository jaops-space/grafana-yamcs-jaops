package source

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
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

func TestMultiplexerUsesDatasourceSecureSocksProxyForWebSocket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	targetCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		greeting := make([]byte, 3)
		if _, err := io.ReadFull(conn, greeting); err != nil {
			errCh <- err
			return
		}
		if !bytes.Equal(greeting, []byte{5, 1, 0}) {
			errCh <- fmt.Errorf("unexpected SOCKS greeting: %v", greeting)
			return
		}
		if _, err := conn.Write([]byte{5, 0}); err != nil {
			errCh <- err
			return
		}

		request := make([]byte, 5)
		if _, err := io.ReadFull(conn, request); err != nil {
			errCh <- err
			return
		}
		if request[0] != 5 || request[1] != 1 || request[3] != 3 {
			errCh <- fmt.Errorf("unexpected SOCKS request: %v", request)
			return
		}
		addressAndPort := make([]byte, int(request[4])+2)
		if _, err := io.ReadFull(conn, addressAndPort); err != nil {
			errCh <- err
			return
		}
		target := string(addressAndPort[:request[4]]) + ":" +
			strconv.Itoa(int(binary.BigEndian.Uint16(addressAndPort[request[4]:])))
		if _, err := conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0}); err != nil {
			errCh <- err
			return
		}
		targetCh <- target
		errCh <- nil
	}()

	cfg := &config.YamcsPluginConfiguration{
		Hosts: map[string]*config.YamcsHostConfiguration{
			"host": {Path: "yamcs.example:8090"},
		},
		Endpoints: map[string]*config.YamcsEndpointConfiguration{},
	}
	opts := httpclient.Options{
		ProxyOptions: &proxy.Options{
			Enabled: true,
			ClientCfg: &proxy.ClientCfg{
				ProxyAddress:  listener.Addr().String(),
				AllowInsecure: true,
			},
		},
	}
	mux, err := NewMultiplexerWithContext(context.Background(), cfg, &config.YamcsSecureConfiguration{}, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Dispose()

	dialContext := mux.Hosts["host"].Client.WebSocket.NetDialContext
	if dialContext == nil {
		t.Fatal("expected WebSocket to use the datasource proxy dialer")
	}
	conn, err := dialContext(context.Background(), "tcp", "yamcs.example:8090")
	if err != nil {
		t.Fatalf("dial through datasource proxy: %v", err)
	}
	defer conn.Close()

	if target := <-targetCh; target != "yamcs.example:8090" {
		t.Fatalf("proxy received target %q", target)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("proxy handshake failed: %v", err)
	}
}
