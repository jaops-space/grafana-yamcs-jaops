package source

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	corehttp "github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// YamcsHost represents a Yamcs server connection along with its instances and processors.
type YamcsHost struct {
	Client     *client.YamcsClient
	Instances  map[string]client.Instance
	Processors map[string]client.Processor
}

// SetupHost sets up a host for live subscriptions.
func (mux *Multiplexer) SetupHost(hostID string) error {
	hostConfig, exists := mux.Configuration.Hosts[hostID]
	if !exists {
		return exception.New(fmt.Sprintf("Configuration for host %s not found", hostID), "CONFIGURATION_NOT_FOUND")
	}

	var tlsConfig corehttp.TLS
	var creds corehttp.Credentials

	if hostConfig.Tls {
		tlsConfig = corehttp.GetTLSConfiguration(!hostConfig.TlsInsecure)
	} else {
		tlsConfig = corehttp.GetNoTLSConfiguration()
	}

	if !hostConfig.Auth {
		creds = &corehttp.NoCredentials{}
	} else {
		username := hostConfig.Username
		secure := mux.GetSecureData(hostID)
		if secure == nil {
			return exception.New(fmt.Sprintf("Secure configuration for host %s not found", hostID), "SECURE_CONFIGURATION_NOT_FOUND")
		}
		password := secure.Password
		creds = &corehttp.BasicAuthCredentials{
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

	mux.Hosts[hostID] = &YamcsHost{
		Client:     yamcsClient,
		Instances:  make(map[string]client.Instance),
		Processors: make(map[string]client.Processor),
	}

	return nil
}

func (mux *Multiplexer) GetSecureData(host string) *config.YamcsSecureHost {
	if host == "" {
		return nil
	}
	secureHost, exists := mux.Secure.Hosts[host]
	if !exists {
		return nil
	}
	return secureHost
}
