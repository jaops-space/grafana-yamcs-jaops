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
	Path     string `json:"path"`
	Tls      bool   `json:"tlsEnabled"`
	Auth     bool   `json:"authEnabled"`
	Protobuf bool   `json:"protobuf"`
}

func ExtractConfig(source backend.DataSourceInstanceSettings) (*YamcsPluginConfiguration, error) {
	configuration := &YamcsPluginConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	return configuration, nil
}
