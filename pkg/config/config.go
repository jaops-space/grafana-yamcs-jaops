package config

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type YamcsPluginConfiguration struct {
	Endpoints map[string]*YamcsEndpointConfiguration `json:"endpoints"`
	Hosts     map[string]*YamcsHostConfiguration     `json:"hosts"`
}

// Validate performs comprehensive validation of the entire plugin configuration.
// Returns a single error with detailed context if anything is invalid.
func (y *YamcsPluginConfiguration) Validate() error {
	if y == nil {
		return fmt.Errorf("configuration is nil")
	}

	var errs []string

	if y.Hosts == nil {
		errs = append(errs, "hosts configuration is missing (null)")
	} else if len(y.Hosts) == 0 {
		errs = append(errs, "hosts configuration is empty – at least one host is required")
	}

	if y.Endpoints == nil {
		errs = append(errs, "endpoints configuration is missing (null)")
	} else if len(y.Endpoints) == 0 {
		errs = append(errs, "endpoints configuration is empty – at least one endpoint is required")
	}

	// Validate each host individually
	for _, host := range y.Hosts {
		if err := host.Validate(y); err != nil {
			hostName := host.Name
			if hostName == "" {
				hostName = "unknown"
			}
			errs = append(errs, fmt.Sprintf("host[%s]: %v", hostName, err))
		}
	}

	// Validate each endpoint individually + cross-check host existence
	for id, ep := range y.Endpoints {
		if err := ep.Validate(y); err != nil {
			errs = append(errs, fmt.Sprintf("endpoint[%s]: %v", id, err))
		}

		if _, exists := y.Hosts[ep.Host]; !exists {
			errs = append(errs, fmt.Sprintf("endpoint[%s]: references unknown host '%s'", id, ep.Host))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// ---------------------------------------------------------------------
// Individual struct validation
// ---------------------------------------------------------------------

// Validate checks a single host configuration
func (h *YamcsHostConfiguration) Validate(y *YamcsPluginConfiguration) error {
	if h == nil {
		return fmt.Errorf("host config is nil")
	}

	var errs []string

	if strings.TrimSpace(h.Path) == "" {
		errs = append(errs, "path is required")
	} else if !isValidHostPort(h.Path) {
		errs = append(errs, fmt.Sprintf("path has invalid URL format: %s", h.Path))
	}

	if h.Auth {
		if strings.TrimSpace(h.Username) == "" {
			errs = append(errs, "username is required when authEnabled=true")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid host config: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (e *YamcsEndpointConfiguration) Validate(y *YamcsPluginConfiguration) error {
	if e == nil {
		return fmt.Errorf("endpoint config is nil")
	}

	var errs []string

	if strings.TrimSpace(e.Host) == "" {
		errs = append(errs, "host reference is required")
	}
	if strings.TrimSpace(e.Instance) == "" {
		errs = append(errs, "instance is required")
	}
	if strings.TrimSpace(e.Processor) == "" {
		errs = append(errs, "processor is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid endpoint config: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ---------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------

// isValidYamcsURL does basic sanity checks on Yamcs base URL/path
func isValidHostPort(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	if !strings.Contains(s, ":") {
		return false
	}

	// Split into host and port
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return false // malformed like "bad:port:extra", ":badport", etc.
	}

	// Host can be empty (means "all interfaces")
	if host != "" {
		if net.ParseIP(host) == nil && !isValidHostname(host) {
			return false
		}
	}

	// Port must be numeric and in valid range
	if port, err := net.LookupPort("tcp", portStr); err != nil || port < 0 || port > 65535 || portStr == "" {
		return false
	}

	return true
}

func isValidHostname(h string) bool {
	// Simple but effective: length, allowed chars, no IP-like patterns already caught
	if len(h) > 255 || len(h) == 0 {
		return false
	}
	return hostnameRegex.MatchString(h)
}

// Pre-compiled regex for hostname validation (RFC 1123 compliant enough for our needs)
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9-]{1,63}\.)*[a-zA-Z0-9-]{1,63}$`)

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
	backend.Logger.Info("ExtractConfig received JSONData",
		"jsonDataString", string(source.JSONData))

	configuration := &YamcsPluginConfiguration{}
	secure := &YamcsSecureConfiguration{}
	err := json.Unmarshal(source.JSONData, configuration)

	if err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	backend.Logger.Info("ExtractConfig unmarshaled config",
		"endpointCount", len(configuration.Endpoints))

	// Extract secure fields for hosts
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
