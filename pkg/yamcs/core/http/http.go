package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
)

// HTTPManager represents a connection to a Yamcs server
type HTTPManager struct {
	URL      string
	AuthRoot string
	APIRoot  string
	Client   *http.Client
	Headers  map[string]string
	// Query holds long-term/persistent query parameters only (e.g. set once
	// at startup and expected to apply to every request going forward).
	// It must never be mutated directly - use SetPersistentQueryParam /
	// DeletePersistentQueryParam, which take queryMu. Per-request/variable
	// query parameters (time ranges, filters, pagination tokens, etc.) must
	// instead be passed directly to the calling method and merged with this
	// map at request-build time in sendRequest - see the query parameter on
	// sendRequest/ProtoRequest/...WithQuery methods.
	Query         map[string]string
	queryMu       sync.RWMutex
	Credentials   Credentials
	UsingProtobuf bool
	OnTokenUpdate func(Credentials)

	RefreshStop     chan struct{} // Channel to stop the refresh ticker
	RefreshInterval time.Duration
	refreshMu       sync.Mutex
}

// SetPersistentQueryParam thread-safely sets a long-term query parameter
// that will be applied to every subsequent request made with this manager
// (merged with any per-request query parameters at request-build time).
func (m *HTTPManager) SetPersistentQueryParam(key, value string) {
	m.queryMu.Lock()
	defer m.queryMu.Unlock()
	m.Query[key] = value
}

// DeletePersistentQueryParam thread-safely removes a long-term query
// parameter previously set via SetPersistentQueryParam.
func (m *HTTPManager) DeletePersistentQueryParam(key string) {
	m.queryMu.Lock()
	defer m.queryMu.Unlock()
	delete(m.Query, key)
}

// snapshotPersistentQuery returns a copy of the persistent query map, safe
// to range over without holding queryMu.
func (m *HTTPManager) snapshotPersistentQuery() map[string]string {
	m.queryMu.RLock()
	defer m.queryMu.RUnlock()
	snapshot := make(map[string]string, len(m.Query))
	for k, v := range m.Query {
		snapshot[k] = v
	}
	return snapshot
}

type requestOptions struct {
	applyCredentials bool
	headers          map[string]string
}

// NewHTTPManager initializes a new Yamcs HTTPManager.
// If an existing *http.Client is provided, it will be used directly (e.g. one
// created via the Grafana plugin SDK). Otherwise a new SDK-based client is
// created with recommended timeouts and middlewares.
func NewHTTPManager(address string, tlsConfig TLS, credentials Credentials, userAgent string, keepAlive bool, protobuf bool, existingClient *http.Client) (*HTTPManager, error) {
	return NewHTTPManagerWithContext(context.Background(), address, tlsConfig, credentials, userAgent, keepAlive, protobuf, existingClient)
}

func NewHTTPManagerWithContext(ctx context.Context, address string, tlsConfig TLS, credentials Credentials, userAgent string, keepAlive bool, protobuf bool, existingClient *http.Client) (*HTTPManager, error) {
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
		URL:             url,
		AuthRoot:        authRoot,
		APIRoot:         apiRoot,
		Client:          httpClient,
		Headers:         make(map[string]string),
		Query:           make(map[string]string),
		Credentials:     credentials,
		UsingProtobuf:   protobuf,
		RefreshInterval: 30 * time.Second,
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
		if err := credentials.Login(ctx, manager); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

// SendRequest sends an HTTP request and automatically applies credentials.
func (m *HTTPManager) SendRequest(ctx context.Context, method string, url string, body []byte) ([]byte, error) {
	return m.sendRequest(ctx, method, url, body, nil, requestOptions{applyCredentials: true})
}

// SendRequestWithQuery is identical to SendRequest but additionally merges
// the given per-request query parameters with the manager's persistent
// Query map. Per-request parameters win on key conflicts.
func (m *HTTPManager) SendRequestWithQuery(ctx context.Context, method string, url string, body []byte, query map[string]string) ([]byte, error) {
	return m.sendRequest(ctx, method, url, body, query, requestOptions{applyCredentials: true})
}

func (m *HTTPManager) sendRequest(ctx context.Context, method string, url string, body []byte, query map[string]string, opts requestOptions) ([]byte, error) {
	if opts.applyCredentials && m.Credentials != nil && m.Credentials.IsExpired() {
		if err := m.Credentials.Refresh(ctx, m); err != nil {
			return nil, err
		}
	}

	reader := bytes.NewReader(body)
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Close = true

	// Apply default headers
	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}

	// Apply query parameters: persistent (long-term) params first, then
	// per-request params, which win on conflict. Both maps are read-only
	// snapshots/caller-owned copies at this point, so no locking is needed
	// here beyond snapshotPersistentQuery's own internal lock.
	q := req.URL.Query()
	for k, v := range m.snapshotPersistentQuery() {
		q.Set(k, v)
	}
	for k, v := range query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// Apply credentials
	if opts.applyCredentials && m.Credentials != nil {
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

// SendJSONRequest sends a JSON HTTP request.
func (m *HTTPManager) SendJSONRequest(ctx context.Context, method string, url string, body any, unmarshalTo any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	respBody, err := m.SendRequest(ctx, method, url, jsonBody)
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

// SendFormRequest sends a form-encoded HTTP request.
func (m *HTTPManager) SendFormRequest(ctx context.Context, method string, url string, form url.Values, headers map[string]string, unmarshalTo any) error {
	reqBody := []byte(form.Encode())
	requestHeaders := make(map[string]string, len(headers)+2)
	requestHeaders["Content-Type"] = "application/x-www-form-urlencoded"
	requestHeaders["Accept"] = "application/json"
	for k, v := range headers {
		requestHeaders[k] = v
	}

	respBody, err := m.sendRequest(ctx, method, url, reqBody, nil, requestOptions{
		applyCredentials: false,
		headers:          requestHeaders,
	})
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
