package config

// ServiceParams is structure for handling user's parameters that
// do not affect the final result of enzyme's commands
type ServiceParams struct {
	SocksProxyHost string
	SocksProxyPort int
}
