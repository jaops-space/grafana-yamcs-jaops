package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	HTTP "net/http"
	"os"
	"strings"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/auth"
)

// HTTPManager represents a connection to a Yamcs server, encapsulating relevant
// details like URLs, headers, and authentication state.
type HTTPManager struct {
	url                    string
	authRoot               string
	apiRoot                string
	Client                 *HTTP.Client
	headers                map[string]string
	Query                  map[string]string
	currentAuthCredentials *auth.AuthCredentials
	usingProtobuf          bool
}

/*
NewContext initializes a new Yamcs connection HTTP.
It configures the URLs and HTTP client based on whether the connection is over TLS.

Parameters:
- address: Server address.
- tlsConfig: Configuration for TLS settings.
- credentials: Account credentials for authentication.
- userAgent: User-Agent string for HTTP requests.
- keepAlive: Whether to maintain persistent connections.
- protobuf: Whether to use protobuf for request/response encoding.

Returns:
- A new HTTPManager object and any error encountered during initialization.
*/
func NewHTTPManager(address string, tlsConfig auth.TLS, credentials auth.AccountCredentials, userAgent string, keepAlive bool, protobuf bool) (*HTTPManager, error) {
	address = strings.TrimSuffix(address, "/")

	var url, authRoot, apiRoot string
	var httpClient *HTTP.Client

	// Setting up TLS configurations if enabled
	if tlsConfig.Enabled {
		var err error
		httpClient, err = setupTLSClient(tlsConfig)
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("https://%s", address)
		authRoot = fmt.Sprintf("https://%s/auth", address)
		apiRoot = fmt.Sprintf("https://%s/api", address)
	} else {
		httpClient = &HTTP.Client{}
		url = fmt.Sprintf("HTTP://%s", address)
		authRoot = fmt.Sprintf("HTTP://%s/auth", address)
		apiRoot = fmt.Sprintf("HTTP://%s/api", address)
	}

	// Creating the HTTP with default headers
	manager := &HTTPManager{
		url, authRoot, apiRoot, httpClient, make(map[string]string), make(map[string]string), nil, protobuf,
	}

	// Setting headers for content type and accept headers
	if protobuf {
		manager.headers["Content-Type"] = "application/protobuf"
		manager.headers["Accept"] = "application/protobuf"
	} else {
		manager.headers["Content-Type"] = "application/json"
		manager.headers["Accept"] = "application/json"
	}

	// Set User-Agent header
	if userAgent == "" {
		manager.headers["User-Agent"] = "jaops-yamcs-go-client"
	} else {
		manager.headers["User-Agent"] = userAgent
	}

	// Attempting login with the provided credentials
	if credentials.Username != "" && credentials.Password != "" {
		authCreds, err := manager.Login(&credentials)
		if err != nil {
			return nil, err
		}
		manager.currentAuthCredentials = authCreds
	}

	return manager, nil
}

/*
setupTLSClient configures an HTTP client with TLS settings.

Parameters:
- tlsConfig: Configuration for TLS settings.

Returns:
- Configured HTTP client and any error encountered.
*/
func setupTLSClient(tlsConfig auth.TLS) (*HTTP.Client, error) {
	var httpClient *HTTP.Client

	if tlsConfig.CertificatePath == "" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: !tlsConfig.Verification,
		}
		transport := &HTTP.Transport{
			TLSClientConfig: tlsConfig,
		}
		httpClient = &HTTP.Client{
			Transport: transport,
		}
	} else {
		caFile := tlsConfig.CertificatePath
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, exception.Wrap("Could not read TLS Cerficate Authority file.", "TLS_ERROR", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: !tlsConfig.Verification,
		}

		transport := &HTTP.Transport{
			TLSClientConfig: tlsConfig,
		}

		httpClient = &HTTP.Client{
			Transport: transport,
		}
	}

	return httpClient, nil
}

/*
SendRequest sends a buffer over to the API and returns the body

Parameters:
- method: HTTP method (GET, POST, PUT, DELETE, etc.)
- url: URL for the request.
- body: Request body to be sent as JSON.

Returns: Body buffer and any error that occured.
*/
func (httpManager *HTTPManager) SendRequest(method string, url string, body []byte) ([]byte, error) {

	reader := bytes.NewReader(body)

	request, reqErr := HTTP.NewRequest(method, url, reader)

	request.Close = true

	if reqErr != nil {
		return []byte{}, reqErr
	}

	for key, value := range httpManager.headers {
		request.Header.Set(key, value)
	}

	values := request.URL.Query()
	for key, value := range httpManager.Query {
		values.Set(key, value)
	}
	request.URL.RawQuery = values.Encode()
	httpManager.Query = make(map[string]string)

	response, respErr := httpManager.Client.Do(request)

	if respErr != nil {
		return nil, respErr
	}

	defer response.Body.Close()

	responseBody, readErr := io.ReadAll(response.Body)

	if readErr != nil {
		return nil, readErr
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return responseBody, exception.New(fmt.Sprintf("Status code was %d", response.StatusCode), "HTTP_STATUS_NOT_OK")
	}

	return responseBody, nil

}

/*
SendJSONRequest is a helper function for sending a raw JSON request for API calls not related to Yamcs, like authentication

Parameters:
- method: HTTP method (GET, POST, PUT, DELETE, etc.)
- url: URL for the request.
- body: Request body to be sent as JSON.
- unmarshallTo: what to unmarshallTo

Returns: Any error encountered during the request or unmarshalling.
*/
func (httpManager *HTTPManager) SendJSONRequest(method string, url string, body any, unmarshalTo any) error {

	jsonBody, jsonErr := json.Marshal(body)
	if jsonErr != nil {
		return jsonErr
	}
	reader := bytes.NewReader(jsonBody)

	request, reqErr := HTTP.NewRequest(method, url, reader)

	request.Close = true

	if reqErr != nil {
		return reqErr
	}

	for key, value := range httpManager.headers {
		request.Header.Add(key, value)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, respErr := httpManager.Client.Do(request)

	if respErr != nil {
		return respErr
	}

	defer response.Body.Close()

	responseBody, readErr := io.ReadAll(response.Body)

	if readErr != nil {
		return readErr
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("HTTP Error, status code: %d, body: %s", response.StatusCode, string(responseBody))
	}

	jsonErr = json.Unmarshal(responseBody, unmarshalTo)

	if jsonErr != nil {
		return jsonErr
	}

	return nil

}
