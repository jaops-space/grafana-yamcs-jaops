package source

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// ConnectionManager manages YAMCS host connections.
type ConnectionManager struct {
	Hosts         map[string]*YamcsHost
	Configuration *config.YamcsPluginConfiguration
	Secure        *config.YamcsSecureConfiguration
	HTTPClient    *http.Client // Shared SDK-provided HTTP client for connection reuse
	SyncMux       sync.Mutex
}

// YamcsHost represents a Yamcs server connection along with its instances and processors.
type YamcsHost struct {
	Client     *client.YamcsClient
	Instances  map[string]client.Instance
	Processors map[string]client.Processor
}

// NewConnectionManager creates a fresh ConnectionManager.
func NewConnectionManager(cfg *config.YamcsPluginConfiguration) *ConnectionManager {
	return &ConnectionManager{
		Hosts:         make(map[string]*YamcsHost),
		Configuration: cfg,
		SyncMux:       sync.Mutex{},
	}
}

// SetupHost configures and connects a Yamcs host.
func (cm *ConnectionManager) SetupHost(hostID string) error {
	hostConfig, exists := cm.Configuration.Hosts[hostID]
	if !exists {
		return exception.New(fmt.Sprintf("Configuration for host %s not found", hostID), "CONFIGURATION_NOT_FOUND")
	}

	var tlsConfig corehttp.TLS
	var creds corehttp.Credentials

	if hostConfig.Tls {
		tlsConfig = corehttp.GetTLSConfiguration(false)
	} else {
		tlsConfig = corehttp.GetNoTLSConfiguration()
	}

	if !hostConfig.Auth {
		creds = &corehttp.NoCredentials{}
	} else {
		username := hostConfig.Username
		secure := cm.GetSecureData(hostID)
		if secure == nil {
			return exception.New(fmt.Sprintf("Secure configuration for host %s not found", hostID), "SECURE_CONFIGURATION_NOT_FOUND")
		}
		password := secure.Password
		creds = &corehttp.BasicAuthCredentials{
			Username: username,
			Password: password,
		}
	}

	// Pass the shared HTTP client so connections are reused across queries
	var opts []client.YamcsClientOption
	if cm.HTTPClient != nil {
		opts = append(opts, client.OptionSetHTTPClient(cm.HTTPClient))
	}

	yamcsClient, err := client.NewYamcsClient(hostConfig.Path, tlsConfig, creds, opts...)
	if err != nil {
		return err
	}

	if err = yamcsClient.EstablishWebSocketConnection(); err != nil {
		return err
	}

	cm.Hosts[hostID] = &YamcsHost{
		Client:     yamcsClient,
		Instances:  make(map[string]client.Instance),
		Processors: make(map[string]client.Processor),
	}

	return nil
}

// GetSecureData retrieves secure host configuration.
func (cm *ConnectionManager) GetSecureData(host string) *config.YamcsSecureHost {
	if host == "" {
		return nil
	}
	if cm.Secure == nil {
		return nil
	}
	secureHost, exists := cm.Secure.Hosts[host]
	if !exists {
		return nil
	}
	return secureHost
}

// GetClient gets or creates a YamcsClient for the given host ID.
func (cm *ConnectionManager) GetClient(hostID string) (*client.YamcsClient, error) {
	cm.SyncMux.Lock()
	defer cm.SyncMux.Unlock()

	host, exists := cm.Hosts[hostID]
	if !exists {
		if err := cm.SetupHost(hostID); err != nil {
			return nil, err
		}
		host = cm.Hosts[hostID]
	}

	if host.Client == nil {
		return nil, exception.New("Unexpected error retrieving Yamcs client", "CONNECTION_CLIENT_NOT_FOUND")
	}

	if !host.Client.IsWebSocketConnected() {
		if err := host.Client.EstablishWebSocketConnection(); err != nil {
			return nil, err
		}
	}

	return host.Client, nil
}

// Dispose shuts down all Yamcs clients (stopping auto-refresh goroutines and
// closing WebSocket connections) and releases host resources.
func (cm *ConnectionManager) Dispose() {
	for _, host := range cm.Hosts {
		if host.Client != nil {
			host.Client.Close()
		}
	}
	cm.Hosts = make(map[string]*YamcsHost)
}
