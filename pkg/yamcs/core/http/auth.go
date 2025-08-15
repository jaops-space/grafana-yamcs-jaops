package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// Credentials interface for all auth types
type Credentials interface {
	Login(*HTTPManager) error
	BeforeRequest(*http.Request) error
	IsExpired() bool
	Refresh(*HTTPManager) error
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
func GetTLSConfiguration(verification bool, certificatePath string) TLS {
	return TLS{
		Enabled:         true,            // TLS is enabled.
		Verification:    verification,    // Set TLS verification as specified.
		CertificatePath: certificatePath, // Path to the certificate if needed.
	}
}

type NoCredentials struct{}

func (n *NoCredentials) Login(*HTTPManager) error {
	return nil // No login required for no credentials.
}
func (n *NoCredentials) BeforeRequest(*http.Request) error {
	return nil // No additional headers needed for no credentials.
}
func (n *NoCredentials) IsExpired() bool {
	return false // No expiration for no credentials.
}
func (n *NoCredentials) Refresh(*HTTPManager) error {
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

// --- Conversion functions ---

func ConvertUserCredentials(manager *HTTPManager, username, password, refreshToken string) (*BearerCredentials, error) {
	data := map[string]string{}
	if username != "" && password != "" {
		data["grant_type"] = "password"
		data["username"] = username
		data["password"] = password
	} else if refreshToken != "" {
		data["grant_type"] = "refresh_token"
		data["refresh_token"] = refreshToken
	} else {
		return nil, fmt.Errorf("either username/password or refresh token required")
	}

	var resp map[string]any
	if err := manager.SendJSONRequest("POST", manager.AuthRoot+"/token", data, &resp); err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}

	expiresIn := int(resp["expires_in"].(float64))
	return &BearerCredentials{
		AccessToken:  resp["access_token"].(string),
		RefreshToken: resp["refresh_token"].(string),
		Expiry:       time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func ConvertServiceAccountCredentials(manager *HTTPManager, clientID, clientSecret, become string) (*ServiceAccountCredentials, error) {
	if clientID == "" || clientSecret == "" || become == "" {
		return nil, fmt.Errorf("client_id, client_secret, and become required")
	}

	data := map[string]string{
		"grant_type": "client_credentials",
		"become":     become,
	}

	var resp map[string]any
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	manager.Headers["Authorization"] = "Basic " + auth
	if err := manager.SendJSONRequest("POST", manager.AuthRoot+"/token", data, &resp); err != nil {
		return nil, fmt.Errorf("service account token request failed: %w", err)
	}

	expiresIn := 0
	if val, ok := resp["expires_in"].(float64); ok {
		expiresIn = int(val)
	}

	return &ServiceAccountCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Become:       become,
		BearerCredentials: BearerCredentials{
			AccessToken: resp["access_token"].(string),
			Expiry:      time.Now().Add(time.Duration(expiresIn) * time.Second),
		},
	}, nil
}

// --- Methods for BearerCredentials ---

func (b *BearerCredentials) Login(manager *HTTPManager) error {
	if b.AccessToken != "" && !b.IsExpired() {
		return nil
	}
	return b.Refresh(manager)
}

func (b *BearerCredentials) Refresh(manager *HTTPManager) error {
	if b.RefreshToken != "" {
		// Refresh using the refresh token
		newCreds, err := ConvertUserCredentials(manager, b.Username, b.Password, b.RefreshToken)
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
		newCreds, err := ConvertUserCredentials(manager, b.Username, b.Password, "")
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
	if b.IsExpired() {
		return b.Refresh(nil) // optionally pass HTTPManager if needed
	}
	if b.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+b.AccessToken)
	}
	return nil
}

// --- Methods for ServiceAccountCredentials ---

func (s *ServiceAccountCredentials) Login(manager *HTTPManager) error {
	return s.Refresh(manager)
}

func (s *ServiceAccountCredentials) Refresh(manager *HTTPManager) error {
	newCreds, err := ConvertServiceAccountCredentials(manager, s.ClientID, s.ClientSecret, s.Become)
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

func (b *BasicAuthCredentials) Login(manager *HTTPManager) error   { return nil }
func (b *BasicAuthCredentials) Refresh(manager *HTTPManager) error { return nil }
func (b *BasicAuthCredentials) IsExpired() bool                    { return false }
func (b *BasicAuthCredentials) BeforeRequest(req *http.Request) error {
	auth := base64.StdEncoding.EncodeToString([]byte(b.Username + ":" + b.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	return nil
}

// --- Methods for APIKeyCredentials ---

func (a *APIKeyCredentials) Login(manager *HTTPManager) error   { return nil }
func (a *APIKeyCredentials) Refresh(manager *HTTPManager) error { return nil }
func (a *APIKeyCredentials) IsExpired() bool                    { return false }
func (a *APIKeyCredentials) BeforeRequest(req *http.Request) error {
	req.Header.Set("x-api-key", a.Key)
	return nil
}

func (m *HTTPManager) StartAutoRefresh() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if m.Credentials.IsExpired() {
					if err := m.Credentials.Refresh(m); err != nil {
						fmt.Printf("failed to refresh token: %v\n", err)
					}
				}
			case <-m.RefreshStop:
				return
			}
		}
	}()
}
