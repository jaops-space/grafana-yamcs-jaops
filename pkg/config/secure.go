package config

type YamcsSecureConfiguration struct {
	Hosts map[string]*YamcsSecureHost
}

type YamcsSecureHost struct {
	Password string
}
