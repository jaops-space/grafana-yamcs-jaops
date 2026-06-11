package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

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
