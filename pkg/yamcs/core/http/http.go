package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

	RefreshStop chan struct{} // Channel to stop the refresh ticker
}

// NewHTTPManager initializes a new Yamcs HTTPManager
func NewHTTPManager(address string, tlsConfig TLS, credentials Credentials, userAgent string, keepAlive bool, protobuf bool) (*HTTPManager, error) {
	address = strings.TrimSuffix(address, "/")

	var url, authRoot, apiRoot string
	var httpClient *http.Client
	var err error

	if tlsConfig.Enabled {
		httpClient, err = setupTLSClient(tlsConfig)
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("https://%s", address)
		authRoot = fmt.Sprintf("https://%s/auth", address)
		apiRoot = fmt.Sprintf("https://%s/api", address)
	} else {
		httpClient = &http.Client{}
		url = fmt.Sprintf("http://%s", address)
		authRoot = fmt.Sprintf("http://%s/auth", address)
		apiRoot = fmt.Sprintf("http://%s/api", address)
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

func setupTLSClient(tlsConfig TLS) (*http.Client, error) {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !tlsConfig.Verification},
		},
	}, nil
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
