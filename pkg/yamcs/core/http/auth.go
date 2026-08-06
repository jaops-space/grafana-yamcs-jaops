package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Credentials interface for all auth types
type Credentials interface {
	Login(context.Context, *HTTPManager) error
	BeforeRequest(*http.Request) error
	IsExpired() bool
	Refresh(context.Context, *HTTPManager) error
}

// TLS represents the configuration for a TLS (Transport Layer Security) connection.
type TLS struct {
	Enabled         bool   // Whether TLS is enabled for the connection.
	Verification    bool   // Whether TLS certificate verification is enabled.
	CertificatePath string // The path to the TLS certificate file (if TLS is enabled).
}

func GetNoTLSConfiguration() TLS {
	return TLS{
		Enabled:         false, // TLS is disabled by default.
		Verification:    false, // No certificate verification.
		CertificatePath: "",    // No certificate path needed.
	}
}

// GetTLSConfiguration returns a TLS configuration with the specified verification setting
// and certificate path for secure connections.
func GetTLSConfiguration(verification bool) TLS {
	return TLS{
		Enabled:      true,         // TLS is enabled.
		Verification: verification, // Set TLS verification as specified.
	}
}

type NoCredentials struct{}

func (n *NoCredentials) Login(context.Context, *HTTPManager) error {
	return nil // No login required for no credentials.
}
func (n *NoCredentials) BeforeRequest(*http.Request) error {
	return nil // No additional headers needed for no credentials.
}
func (n *NoCredentials) IsExpired() bool {
	return false // No expiration for no credentials.
}
func (n *NoCredentials) Refresh(context.Context, *HTTPManager) error {
	return nil // No refresh needed for no credentials.
}

// BearerCredentials represents username/password or token-based credentials
type BearerCredentials struct {
	Username     string
	Password     string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time

	onTokenUpdate func(creds *BearerCredentials)
}

// ServiceAccountCredentials represents client credentials + "become" impersonation
type ServiceAccountCredentials struct {
	ClientID     string
	ClientSecret string
	Become       string
	BearerCredentials
}

// BasicAuthCredentials represents simple username/password Basic Auth
type BasicAuthCredentials struct {
	Username string
	Password string
}

// APIKeyCredentials represents API key authentication
type APIKeyCredentials struct {
	Key string
}

func getTokenString(resp map[string]any, key string) (string, error) {
	value, exists := resp[key]
	if !exists {
		return "", fmt.Errorf("token response missing %s", key)
	}

	strValue, ok := value.(string)
	if !ok || strValue == "" {
		return "", fmt.Errorf("token response field %s is invalid", key)
	}

	return strValue, nil
}

func getTokenExpirySeconds(resp map[string]any) (int, error) {
	value, exists := resp["expires_in"]
	if !exists {
		return 0, fmt.Errorf("token response missing expires_in")
	}

	seconds, ok := value.(float64)
	if !ok || seconds <= 0 {
		return 0, fmt.Errorf("token response field expires_in is invalid")
	}

	return int(seconds), nil
}

// --- Conversion functions ---

func ConvertUserCredentials(ctx context.Context, manager *HTTPManager, username, password, refreshToken string) (*BearerCredentials, error) {
	form := url.Values{}
	if username != "" && password != "" {
		form.Set("grant_type", "password")
		form.Set("username", username)
		form.Set("password", password)
	} else if refreshToken != "" {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", refreshToken)
	} else {
		return nil, fmt.Errorf("either username/password or refresh token required")
	}

	var resp map[string]any
	if err := manager.SendFormRequest(ctx, "POST", manager.AuthRoot+"/token", form, nil, &resp); err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}

	expiresIn, err := getTokenExpirySeconds(resp)
	if err != nil {
		return nil, err
	}

	accessToken, err := getTokenString(resp, "access_token")
	if err != nil {
		return nil, err
	}

	refreshTokenValue, err := getTokenString(resp, "refresh_token")
	if err != nil {
		return nil, err
	}

	return &BearerCredentials{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenValue,
		Expiry:       time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func ConvertServiceAccountCredentials(ctx context.Context, manager *HTTPManager, clientID, clientSecret, become string) (*ServiceAccountCredentials, error) {
	if clientID == "" || clientSecret == "" || become == "" {
		return nil, fmt.Errorf("client_id, client_secret, and become required")
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("become", become)

	var resp map[string]any
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	headers := map[string]string{"Authorization": "Basic " + auth}
	if err := manager.SendFormRequest(ctx, "POST", manager.AuthRoot+"/token", form, headers, &resp); err != nil {
		return nil, fmt.Errorf("service account token request failed: %w", err)
	}

	expiresIn, err := getTokenExpirySeconds(resp)
	if err != nil {
		return nil, err
	}

	accessToken, err := getTokenString(resp, "access_token")
	if err != nil {
		return nil, err
	}

	return &ServiceAccountCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Become:       become,
		BearerCredentials: BearerCredentials{
			AccessToken: accessToken,
			Expiry:      time.Now().Add(time.Duration(expiresIn) * time.Second),
		},
	}, nil
}

// --- Methods for BearerCredentials ---

func (b *BearerCredentials) Login(ctx context.Context, manager *HTTPManager) error {
	if b.AccessToken != "" && !b.IsExpired() {
		return nil
	}
	return b.Refresh(ctx, manager)
}

func (b *BearerCredentials) Refresh(ctx context.Context, manager *HTTPManager) error {
	if b.RefreshToken != "" {
		// Refresh using the refresh token
		newCreds, err := ConvertUserCredentials(ctx, manager, "", "", b.RefreshToken)
		if err != nil {
			return err
		}
		b.AccessToken = newCreds.AccessToken
		b.RefreshToken = newCreds.RefreshToken
		b.Expiry = newCreds.Expiry
		if b.onTokenUpdate != nil {
			b.onTokenUpdate(newCreds)
		}
		return nil
	} else if b.Username != "" && b.Password != "" {
		// Refresh using username/password
		newCreds, err := ConvertUserCredentials(ctx, manager, b.Username, b.Password, "")
		if err != nil {
			return err
		}
		b.AccessToken = newCreds.AccessToken
		b.RefreshToken = newCreds.RefreshToken
		b.Expiry = newCreds.Expiry
		if b.onTokenUpdate != nil {
			b.onTokenUpdate(newCreds)
		}
		return nil
	}
	return fmt.Errorf("no credentials available for refresh")
}

func (b *BearerCredentials) IsExpired() bool {
	if b.Expiry.IsZero() {
		return false
	}
	return time.Now().After(b.Expiry)
}

func (b *BearerCredentials) BeforeRequest(req *http.Request) error {
	if b.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+b.AccessToken)
	}
	return nil
}

// --- Methods for ServiceAccountCredentials ---

func (s *ServiceAccountCredentials) Login(ctx context.Context, manager *HTTPManager) error {
	return s.Refresh(ctx, manager)
}

func (s *ServiceAccountCredentials) Refresh(ctx context.Context, manager *HTTPManager) error {
	newCreds, err := ConvertServiceAccountCredentials(ctx, manager, s.ClientID, s.ClientSecret, s.Become)
	if err != nil {
		return err
	}
	s.AccessToken = newCreds.AccessToken
	s.Expiry = newCreds.Expiry
	if s.onTokenUpdate != nil {
		s.onTokenUpdate(&newCreds.BearerCredentials)
	}
	return nil
}

func (s *ServiceAccountCredentials) IsExpired() bool {
	return s.BearerCredentials.IsExpired()
}

func (s *ServiceAccountCredentials) BeforeRequest(req *http.Request) error {
	return s.BearerCredentials.BeforeRequest(req)
}

// --- Methods for BasicAuthCredentials ---

func (b *BasicAuthCredentials) Login(context.Context, *HTTPManager) error   { return nil }
func (b *BasicAuthCredentials) Refresh(context.Context, *HTTPManager) error { return nil }
func (b *BasicAuthCredentials) IsExpired() bool                             { return false }
func (b *BasicAuthCredentials) BeforeRequest(req *http.Request) error {
	auth := base64.StdEncoding.EncodeToString([]byte(b.Username + ":" + b.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	return nil
}

// --- Methods for APIKeyCredentials ---

func (a *APIKeyCredentials) Login(context.Context, *HTTPManager) error   { return nil }
func (a *APIKeyCredentials) Refresh(context.Context, *HTTPManager) error { return nil }
func (a *APIKeyCredentials) IsExpired() bool                             { return false }
func (a *APIKeyCredentials) BeforeRequest(req *http.Request) error {
	req.Header.Set("x-api-key", a.Key)
	return nil
}

func credentialsCanExpire(credentials Credentials) bool {
	switch creds := credentials.(type) {
	case *BearerCredentials:
		return !creds.Expiry.IsZero()
	case *ServiceAccountCredentials:
		return !creds.Expiry.IsZero()
	default:
		return false
	}
}

func (m *HTTPManager) StartAutoRefresh() {
	m.StopAutoRefresh()
	if m.Credentials == nil || !credentialsCanExpire(m.Credentials) {
		return
	}

	stop := make(chan struct{})
	m.refreshMu.Lock()
	m.RefreshStop = stop
	m.refreshMu.Unlock()

	interval := m.RefreshInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if m.Credentials.IsExpired() {
					if err := m.Credentials.Refresh(context.Background(), m); err != nil {
						backend.Logger.Error("failed to refresh token", "error", err)
					}
				}
			case <-stop:
				return
			}
		}
	}()
}

func (m *HTTPManager) StopAutoRefresh() {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	if m.RefreshStop == nil {
		return
	}

	close(m.RefreshStop)
	m.RefreshStop = nil
}

func (m *HTTPManager) Dispose() {
	m.StopAutoRefresh()
	if m.Client != nil {
		m.Client.CloseIdleConnections()
	}
}
