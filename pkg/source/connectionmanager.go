package source

import (
	"fmt"
	"sync"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// ConnectionManager manages YAMCS host connections.
type ConnectionManager struct {
	Hosts         map[string]*YamcsHost
	Configuration *config.YamcsPluginConfiguration
	Secure        *config.YamcsSecureConfiguration
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

	var tlsConfig http.TLS
	var creds http.Credentials

	if hostConfig.Tls {
		tlsConfig = http.GetTLSConfiguration(false)
	} else {
		tlsConfig = http.GetNoTLSConfiguration()
	}

	if !hostConfig.Auth {
		creds = &http.NoCredentials{}
	} else {
		username := hostConfig.Username
		secure := cm.GetSecureData(hostID)
		if secure == nil {
			return exception.New(fmt.Sprintf("Secure configuration for host %s not found", hostID), "SECURE_CONFIGURATION_NOT_FOUND")
		}
		password := secure.Password
		creds = &http.BasicAuthCredentials{
			Username: username,
			Password: password,
		}
	}

	yamcsClient, err := client.NewYamcsClient(hostConfig.Path, tlsConfig, creds)
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

// Dispose closes all websocket connections.
func (cm *ConnectionManager) Dispose() {
	for _, host := range cm.Hosts {
		if host.Client != nil {
			host.Client.CloseWebSocketConnection()
		}
	}
	cm.Hosts = make(map[string]*YamcsHost)
}
