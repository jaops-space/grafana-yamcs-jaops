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
	Name        string `json:"name"`
	Description string `json:"description"`
	Host        string `json:"host"`
	Instance    string `json:"instance"`
	Processor   string `json:"processor"`
}

type YamcsHostConfiguration struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Tls         bool   `json:"tlsEnabled"`
	TlsInsecure bool   `json:"tlsInsecure"`
	Auth        bool   `json:"authEnabled"`
	Username    string `json:"username"`
	Protobuf    bool   `json:"protobuf"`
}

func ExtractConfig(source backend.DataSourceInstanceSettings) (*YamcsPluginConfiguration, *YamcsSecureConfiguration, error) {
	// Debug: log what Grafana sent us
	backend.Logger.Debug("ExtractConfig received JSONData",
		"jsonDataString", string(source.JSONData))

	configuration := &YamcsPluginConfiguration{}
	secure := &YamcsSecureConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)

	if err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	backend.Logger.Debug("ExtractConfig unmarshaled config",
		"endpointCount", len(configuration.Endpoints))

	// Extract secure fields for hosts
	secure.Hosts = make(map[string]*YamcsSecureHost)
	for hostName, hostConfig := range configuration.Hosts {
		if hostConfig.Auth {
			backend.Logger.Debug("ExtractConfig processing secure config for host",
				"hostName", hostName)
			secure.Hosts[hostName] = &YamcsSecureHost{}
			passwordKey := hostName + "-password"
			if password, ok := source.DecryptedSecureJSONData[passwordKey]; ok {
				secure.Hosts[hostName].Password = password
			} else {
				return nil, nil, fmt.Errorf("missing secure password for host %s", hostName)
			}
		}
	}

	return configuration, secure, nil
}
