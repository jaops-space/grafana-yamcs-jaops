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

func (y *YamcsPluginConfiguration) Validate() error {
	if y.Endpoints == nil {
		return fmt.Errorf("missing endpoints configurations")
	}

	if y.Hosts == nil {
		return fmt.Errorf("missing hosts configuration")
	}

	if len(y.Endpoints) == 0 || len(y.Hosts) == 0 {
		return fmt.Errorf("plugin requires at least one endpoint and one host configuration")
	}

	return nil
}

type YamcsEndpointConfiguration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Host        string `json:"host"`
	Instance    string `json:"instance"`
	Processor   string `json:"processor"`
}

type YamcsHostConfiguration struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Tls      bool   `json:"tlsEnabled"`
	Auth     bool   `json:"authEnabled"`
	Username string `json:"username"`
	Protobuf bool   `json:"protobuf"`
}

func ExtractConfig(source backend.DataSourceInstanceSettings) (*YamcsPluginConfiguration, *YamcsSecureConfiguration, error) {

	configuration := &YamcsPluginConfiguration{}
	secure := &YamcsSecureConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)
	if err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	// Extract secure fields
	// loop through hosts of the configuration YamcsPluginConfiguration, for each host, grab all fields from DecodedSecureJSON that start with the host name, and populate YamcsSecureConfiguration.Hosts[hostname].Password if field ends in -password
	secure.Hosts = make(map[string]*YamcsSecureHost)
	for hostName, hostConfig := range configuration.Hosts {
		if hostConfig.Auth {
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
