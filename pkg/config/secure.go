package config

// YamcsSecureConfiguration holds secrets for hosts
type YamcsSecureConfiguration struct {
	Hosts map[string]*YamcsSecureHost
}

// YamcsSecureHost stores the password for a Yamcs host.
type YamcsSecureHost struct {
	Password string
}

func (scfg *YamcsSecureConfiguration) GetHost(host string) *YamcsSecureHost {
	return scfg.Hosts[host]
}
