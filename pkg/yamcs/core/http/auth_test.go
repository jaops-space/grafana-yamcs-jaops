package http

import (
	"context"
	"encoding/base64"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type authMockTransport struct {
	t *testing.T

	mu        sync.Mutex
	requests  []*stdhttp.Request
	bodies    []string
	responses []string
}

func newAuthMockTransport(t *testing.T, responses ...string) *authMockTransport {
	t.Helper()
	return &authMockTransport{t: t, responses: responses}
}

func (m *authMockTransport) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		m.t.Fatalf("read request body: %v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, req.Clone(req.Context()))
	m.bodies = append(m.bodies, string(body))

	response := `{}`
	if len(m.responses) > 0 {
		response = m.responses[0]
		m.responses = m.responses[1:]
	}

	return &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Header:     stdhttp.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(response)),
		Request:    req,
	}, nil
}

func (m *authMockTransport) requestAt(index int) *stdhttp.Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index >= len(m.requests) {
		m.t.Fatalf("expected request %d, got %d requests", index, len(m.requests))
	}
	return m.requests[index]
}

func (m *authMockTransport) bodyAt(index int) url.Values {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index >= len(m.bodies) {
		m.t.Fatalf("expected body %d, got %d bodies", index, len(m.bodies))
	}
	values, err := url.ParseQuery(m.bodies[index])
	if err != nil {
		m.t.Fatalf("parse form body %q: %v", m.bodies[index], err)
	}
	return values
}

func newTestHTTPManager(t *testing.T, transport stdhttp.RoundTripper, credentials Credentials) *HTTPManager {
	t.Helper()
	manager, err := NewHTTPManager(
		"yamcs.example",
		GetNoTLSConfiguration(),
		credentials,
		"auth-test",
		true,
		false,
		&stdhttp.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("create HTTP manager: %v", err)
	}
	return manager
}

func TestAuthCredentialsApplyRequestHeaders(t *testing.T) {
	tests := []struct {
		name          string
		credentials   Credentials
		wantHeader    string
		wantHeaderVal string
	}{
		{
			name:        "no credentials",
			credentials: &NoCredentials{},
		},
		{
			name:          "basic",
			credentials:   &BasicAuthCredentials{Username: "operator", Password: "secret:with-colon"},
			wantHeader:    "Authorization",
			wantHeaderVal: "Basic " + base64.StdEncoding.EncodeToString([]byte("operator:secret:with-colon")),
		},
		{
			name:          "api key",
			credentials:   &APIKeyCredentials{Key: "api-key-value"},
			wantHeader:    "x-api-key",
			wantHeaderVal: "api-key-value",
		},
		{
			name:          "bearer",
			credentials:   &BearerCredentials{AccessToken: "access-token"},
			wantHeader:    "Authorization",
			wantHeaderVal: "Bearer access-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := newAuthMockTransport(t, `{"ok":true}`)
			manager := newTestHTTPManager(t, transport, tt.credentials)

			if _, err := manager.SendRequest(context.Background(), "GET", manager.APIRoot+"/instances", nil); err != nil {
				t.Fatalf("send request: %v", err)
			}

			req := transport.requestAt(0)
			if tt.wantHeader == "" {
				if got := req.Header.Get("Authorization"); got != "" {
					t.Fatalf("expected no authorization header, got %q", got)
				}
				if got := req.Header.Get("x-api-key"); got != "" {
					t.Fatalf("expected no api key header, got %q", got)
				}
				return
			}

			if got := req.Header.Get(tt.wantHeader); got != tt.wantHeaderVal {
				t.Fatalf("expected %s %q, got %q", tt.wantHeader, tt.wantHeaderVal, got)
			}
		})
	}
}

func TestConvertUserCredentialsUsesPasswordGrantForm(t *testing.T) {
	transport := newAuthMockTransport(t, `{"access_token":"access-1","refresh_token":"refresh-1","expires_in":60}`)
	manager := newTestHTTPManager(t, transport, nil)

	credentials, err := ConvertUserCredentials(context.Background(), manager, "operator", "secret", "")
	if err != nil {
		t.Fatalf("convert user credentials: %v", err)
	}

	if credentials.AccessToken != "access-1" || credentials.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}

	req := transport.requestAt(0)
	if req.Method != "POST" || req.URL.String() != manager.AuthRoot+"/token" {
		t.Fatalf("unexpected token request: %s %s", req.Method, req.URL.String())
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form content type, got %q", got)
	}

	body := transport.bodyAt(0)
	if got := body.Get("grant_type"); got != "password" {
		t.Fatalf("expected password grant, got %q", got)
	}
	if got := body.Get("username"); got != "operator" {
		t.Fatalf("expected username operator, got %q", got)
	}
	if got := body.Get("password"); got != "secret" {
		t.Fatalf("expected password secret, got %q", got)
	}
}

func TestNewHTTPManagerWithContextUsesLoginContext(t *testing.T) {
	type contextKey string

	const key contextKey = "constructor-request-id"
	transport := newAuthMockTransport(t, `{"access_token":"access-login","refresh_token":"refresh-login","expires_in":60}`)
	ctx := context.WithValue(context.Background(), key, "datasource-construction")

	_, err := NewHTTPManagerWithContext(
		ctx,
		"yamcs.example",
		GetNoTLSConfiguration(),
		&BearerCredentials{Username: "operator", Password: "secret"},
		"auth-test",
		true,
		false,
		&stdhttp.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("create HTTP manager: %v", err)
	}

	req := transport.requestAt(0)
	if got := req.Context().Value(key); got != "datasource-construction" {
		t.Fatalf("expected constructor login to use caller context, got %v", got)
	}
}

func TestBearerCredentialsRefreshUsesRefreshTokenAndUpdatesCallback(t *testing.T) {
	transport := newAuthMockTransport(t,
		`{"access_token":"access-2","refresh_token":"refresh-2","expires_in":60}`,
		`{"ok":true}`,
	)
	credentials := &BearerCredentials{
		Username:     "operator",
		Password:     "old-password",
		AccessToken:  "expired-access",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(-time.Second),
	}

	var updated *BearerCredentials
	credentials.onTokenUpdate = func(creds *BearerCredentials) {
		updated = creds
	}

	manager := newTestHTTPManager(t, transport, nil)
	manager.Credentials = credentials

	if _, err := manager.SendRequest(context.Background(), "GET", manager.APIRoot+"/instances", nil); err != nil {
		t.Fatalf("send request: %v", err)
	}

	body := transport.bodyAt(0)
	if got := body.Get("grant_type"); got != "refresh_token" {
		t.Fatalf("expected refresh token grant, got %q", got)
	}
	if got := body.Get("refresh_token"); got != "refresh-1" {
		t.Fatalf("expected refresh token refresh-1, got %q", got)
	}
	if body.Get("username") != "" || body.Get("password") != "" {
		t.Fatalf("refresh token request should not include username/password, got %v", body)
	}
	if credentials.AccessToken != "access-2" || credentials.RefreshToken != "refresh-2" {
		t.Fatalf("credentials were not updated: %#v", credentials)
	}
	if updated == nil || updated.AccessToken != "access-2" || updated.RefreshToken != "refresh-2" {
		t.Fatalf("token update callback did not receive new credentials: %#v", updated)
	}

	req := transport.requestAt(1)
	if got := req.Header.Get("Authorization"); got != "Bearer access-2" {
		t.Fatalf("expected refreshed bearer header, got %q", got)
	}
}

func TestBearerRefreshUsesRequestContext(t *testing.T) {
	type contextKey string

	const key contextKey = "request-id"
	transport := newAuthMockTransport(t,
		`{"access_token":"access-ctx","refresh_token":"refresh-ctx","expires_in":60}`,
		`{"ok":true}`,
	)
	credentials := &BearerCredentials{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(-time.Second),
	}
	manager := newTestHTTPManager(t, transport, nil)
	manager.Credentials = credentials

	ctx := context.WithValue(context.Background(), key, "grafana-request")
	if _, err := manager.SendRequest(ctx, "GET", manager.APIRoot+"/instances", nil); err != nil {
		t.Fatalf("send request: %v", err)
	}

	req := transport.requestAt(0)
	if got := req.Context().Value(key); got != "grafana-request" {
		t.Fatalf("expected refresh request to use caller context, got %v", got)
	}
}

func TestServiceAccountCredentialsUseClientCredentialsGrant(t *testing.T) {
	transport := newAuthMockTransport(t,
		`{"access_token":"service-access","expires_in":60}`,
		`{"ok":true}`,
	)
	manager := newTestHTTPManager(t, transport, nil)

	credentials, err := ConvertServiceAccountCredentials(context.Background(), manager, "client-id", "client-secret", "operator")
	if err != nil {
		t.Fatalf("convert service account credentials: %v", err)
	}
	if credentials.AccessToken != "service-access" {
		t.Fatalf("expected service access token, got %#v", credentials)
	}

	tokenReq := transport.requestAt(0)
	wantBasic := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
	if got := tokenReq.Header.Get("Authorization"); got != wantBasic {
		t.Fatalf("expected token request basic auth %q, got %q", wantBasic, got)
	}

	body := transport.bodyAt(0)
	if got := body.Get("grant_type"); got != "client_credentials" {
		t.Fatalf("expected client credentials grant, got %q", got)
	}
	if got := body.Get("become"); got != "operator" {
		t.Fatalf("expected become operator, got %q", got)
	}

	manager.Credentials = credentials
	if _, err := manager.SendRequest(context.Background(), "GET", manager.APIRoot+"/instances", nil); err != nil {
		t.Fatalf("send request: %v", err)
	}

	apiReq := transport.requestAt(1)
	if got := apiReq.Header.Get("Authorization"); got != "Bearer service-access" {
		t.Fatalf("expected subsequent bearer auth, got %q", got)
	}
}

func TestStartAutoRefreshRefreshesExpiredCredentials(t *testing.T) {
	transport := newAuthMockTransport(t, `{"access_token":"auto-access","refresh_token":"auto-refresh","expires_in":60}`)
	updated := make(chan *BearerCredentials, 1)
	credentials := &BearerCredentials{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(-time.Second),
	}
	credentials.onTokenUpdate = func(creds *BearerCredentials) {
		updated <- creds
	}
	manager := newTestHTTPManager(t, transport, nil)
	manager.Credentials = credentials
	manager.RefreshInterval = time.Millisecond

	manager.StartAutoRefresh()
	defer manager.StopAutoRefresh()

	select {
	case creds := <-updated:
		if creds.AccessToken != "auto-access" || creds.RefreshToken != "auto-refresh" {
			t.Fatalf("unexpected refreshed credentials: %#v", creds)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected auto refresh to update credentials")
	}

	body := transport.bodyAt(0)
	if got := body.Get("grant_type"); got != "refresh_token" {
		t.Fatalf("expected refresh token grant, got %q", got)
	}
}

func TestHTTPManagerDisposeStopsRefreshAndClosesIdleConnections(t *testing.T) {
	transport := &closeTrackingTransport{}
	manager := newTestHTTPManager(t, transport, &BearerCredentials{
		AccessToken: "access",
		Expiry:      time.Now().Add(time.Hour),
	})

	manager.StartAutoRefresh()
	if manager.RefreshStop == nil {
		t.Fatal("expected refresh loop to start")
	}

	manager.Dispose()

	if manager.RefreshStop != nil {
		t.Fatal("expected refresh loop to stop")
	}
	if !transport.closedIdleConnections {
		t.Fatal("expected idle connections to be closed")
	}
}

type closeTrackingTransport struct {
	closedIdleConnections bool
}

func (t *closeTrackingTransport) RoundTrip(*stdhttp.Request) (*stdhttp.Response, error) {
	return &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Header:     stdhttp.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}, nil
}

func (t *closeTrackingTransport) CloseIdleConnections() {
	t.closedIdleConnections = true
}
