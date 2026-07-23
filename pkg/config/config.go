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
}

func (hcfg *YamcsHostConfiguration) DisplayName() string {
	if hcfg.Name != "" {
		return hcfg.Name
	}
	if hcfg.Path != "" {
		return fmt.Sprintf("host at %s", hcfg.Path)
	}
	return "<unnamed host>"
}

func ExtractConfig(source backend.DataSourceInstanceSettings) (*YamcsPluginConfiguration, *YamcsSecureConfiguration, error) {

	// Debug: log what Grafana sent us
	backend.Logger.Debug("ExtractConfig received JSONData",
		"jsonDataString", string(source.JSONData))

	configuration := &YamcsPluginConfiguration{}
	secure := &YamcsSecureConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)

	for hostID, host := range configuration.Hosts {
		host.ID = hostID
	}
	for endpointID, endpoint := range configuration.Endpoints {
		endpoint.ID = endpointID
	}

	if err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
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
