package multiplexer

import "github.com/jaops-space/grafana-yamcs-jaops/pkg/config"

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
