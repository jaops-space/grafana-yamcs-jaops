package config

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type YamcsPluginConfiguration struct {
	Endpoints map[string]*YamcsEndpointConfiguration `json:"endpoints"`
	Hosts     map[string]*YamcsHostConfiguration     `json:"hosts"`
}

type YamcsEndpointConfiguration struct {
	ID          string
	Name        string `json:"name"`
	Description string `json:"description"`
	Host        string `json:"host"`
	Instance    string `json:"instance"`
	Processor   string `json:"processor"`
}

func (epcfg *YamcsEndpointConfiguration) DisplayName() string {
	if epcfg.Name != "" {
		return epcfg.Name
	}
	if epcfg.ID != "" {
		return epcfg.ID
	}
	return "<unnamed endpoint>"
}

type YamcsHostConfiguration struct {
	ID          string
	Name        string `json:"name"`
	Path        string `json:"path"`
	Tls         bool   `json:"tlsEnabled"`
	TlsInsecure bool   `json:"tlsInsecure"`
	Auth        bool   `json:"authEnabled"`
	Username    string `json:"username"`
	Protobuf    bool   `json:"protobuf"`

	// DiscoverInstances makes an endpoint for each instance of the host,
	// on its realtime processor. The endpoint's ID is the instance's name,
	// or its alias if Aliases has one. The endpoints are kept up to date
	// while the datasource runs. An endpoint configured by hand takes
	// precedence over a discovered one with the same ID.
	DiscoverInstances bool `json:"discoverInstances"`
	// Aliases maps an instance name to the ID of its discovered endpoint
	// (e.g. fm1: sat-1), so dashboards made elsewhere work unchanged.
	Aliases map[string]string `json:"aliases"`
}

// EndpointID returns the ID of the endpoint discovered for an instance.
func (hcfg *YamcsHostConfiguration) EndpointID(instance string) string {
	if a, ok := hcfg.Aliases[instance]; ok && a != "" {
		return a
	}
	return instance
}

func (hcfg *YamcsHostConfiguration) DisplayName() string {
	if hcfg.Name != "" {
		return hcfg.Name
	}
	if hcfg.Path != "" {
		if !isValidHostPort(hcfg.Path) {
			return "host at <invalid address>"
		}
		return fmt.Sprintf("host at %s", hcfg.Path)
	}
	return "<unnamed host>"
}

func ExtractConfig(source backend.DataSourceInstanceSettings) (*YamcsPluginConfiguration, *YamcsSecureConfiguration, error) {
	configuration := &YamcsPluginConfiguration{}
	secure := &YamcsSecureConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)
	if err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	for hostID, host := range configuration.Hosts {
		if host == nil {
			return nil, nil, fmt.Errorf("host %q is null", hostID)
		}
		host.ID = hostID
	}
	for endpointID, endpoint := range configuration.Endpoints {
		if endpoint == nil {
			return nil, nil, fmt.Errorf("endpoint %q is null", endpointID)
		}
		endpoint.ID = endpointID
	}

	backend.Logger.Debug("ExtractConfig unmarshaled config",
		"endpointCount", len(configuration.Endpoints))

	// Extract secure fields for hosts
	secure.Hosts = make(map[string]*YamcsSecureHost)
	for hostID, hostConfig := range configuration.Hosts {
		if hostConfig.Auth {
			backend.Logger.Debug("ExtractConfig processing secure config for host",
				"hostName", hostID)
			secure.Hosts[hostID] = &YamcsSecureHost{}
			passwordKey := hostID + "-password"
			if password, ok := source.DecryptedSecureJSONData[passwordKey]; ok {
				secure.Hosts[hostID].Password = password
			} else {
				return nil, nil, fmt.Errorf("missing secure password for host %s", hostID)
			}
		}
	}

	return configuration, secure, nil
}
