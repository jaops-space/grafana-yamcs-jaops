package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

// HTTPManager represents a connection to a Yamcs server
type HTTPManager struct {
	URL           string
	AuthRoot      string
	APIRoot       string
	Client        *http.Client
	Headers       map[string]string
	Query         map[string]string
	Credentials   Credentials
	UsingProtobuf bool
	OnTokenUpdate func(Credentials)

	refreshCancel context.CancelFunc // cancels the current auto-refresh goroutine
	refreshMu     sync.Mutex         // guards refreshCancel
}

// NewHTTPManager initializes a new Yamcs HTTPManager.
// If an existing *http.Client is provided, it will be used directly (e.g. one
// created via the Grafana plugin SDK). Otherwise a new SDK-based client is
// created with recommended timeouts and middlewares.
func NewHTTPManager(address string, tlsConfig TLS, credentials Credentials, userAgent string, keepAlive bool, protobuf bool, existingClient *http.Client) (*HTTPManager, error) {
	address = strings.TrimSuffix(address, "/")

	var url, authRoot, apiRoot string

	// Determine the scheme based on TLS configuration
	scheme := "http"
	if tlsConfig.Enabled {
		scheme = "https"
	}
	url = fmt.Sprintf("%s://%s", scheme, address)
	authRoot = fmt.Sprintf("%s://%s/auth", scheme, address)
	apiRoot = fmt.Sprintf("%s://%s/api", scheme, address)

	// Use the provided client or create one via the Grafana SDK
	httpClient := existingClient
	if httpClient == nil {
		opts := httpclient.Options{}
		if tlsConfig.Enabled {
			opts.TLS = &httpclient.TLSOptions{
				InsecureSkipVerify: !tlsConfig.Verification,
			}
		}
		var err error
		httpClient, err = httpclient.New(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}
	}

	manager := &HTTPManager{
		URL:           url,
		AuthRoot:      authRoot,
		APIRoot:       apiRoot,
		Client:        httpClient,
		Headers:       make(map[string]string),
		Query:         make(map[string]string),
		Credentials:   credentials,
		UsingProtobuf: protobuf,
	}

	if protobuf {
		manager.Headers["Content-Type"] = "application/protobuf"
		manager.Headers["Accept"] = "application/protobuf"
	} else {
		manager.Headers["Content-Type"] = "application/json"
		manager.Headers["Accept"] = "application/json"
	}

	if userAgent == "" {
		manager.Headers["User-Agent"] = "jaops-yamcs-go-client"
	} else {
		manager.Headers["User-Agent"] = userAgent
	}

	if credentials != nil {
		if err := credentials.Login(manager); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

// NewSDKClient creates an *http.Client via the Grafana plugin SDK with the
// given TLS settings. This is the recommended way to obtain a client for
// Grafana data-source plugins because it auto-applies timeouts, keep-alive,
// and observability middlewares.
func NewSDKClient(tlsConfig TLS) (*http.Client, error) {
	opts := httpclient.Options{}
	if tlsConfig.Enabled {
		opts.TLS = &httpclient.TLSOptions{
			InsecureSkipVerify: !tlsConfig.Verification,
		}
	}
	return httpclient.New(opts)
}

// SendRequest sends an HTTP request and automatically applies credentials
func (m *HTTPManager) SendRequest(method string, url string, body []byte) ([]byte, error) {
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	m.Credentials.BeforeRequest(req)

	req.Close = true

	// Apply default headers
	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}

	// Apply query parameters
	q := req.URL.Query()
	for k, v := range m.Query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	m.Query = make(map[string]string)

	// Apply credentials
	if m.Credentials != nil {
		if err := m.Credentials.BeforeRequest(req); err != nil {
			return nil, err
		}
	}

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return respBody, exception.New(fmt.Sprintf("Status code was %d", resp.StatusCode), "HTTP_STATUS_NOT_OK")
	}

	return respBody, nil
}

// SendJSONRequest sends a JSON HTTP request
func (m *HTTPManager) SendJSONRequest(method string, url string, body any, unmarshalTo any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	respBody, err := m.SendRequest(method, url, jsonBody)
	if err != nil {
		return err
	}

	if unmarshalTo != nil {
		if err := json.Unmarshal(respBody, unmarshalTo); err != nil {
			return err
		}
	}

	return nil
}
