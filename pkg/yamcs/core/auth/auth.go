package auth

/*
   Data structures for holding credentials and TLS configuration.
   These are used for managing authentication and secure connections.
*/

// AccountCredentials holds the username and password for account-based authentication.
type AccountCredentials struct {
	Username string // The username for account authentication.
	Password string // The password for account authentication.
}

// AuthCredentials holds the authentication tokens (access and optionally refresh tokens)
// received upon successful login.
type AuthCredentials struct {
	AccessToken  string `json:"access_token"`  // The access token for API requests.
	RefreshToken string `json:"refresh_token"` // The refresh token, if available.
	Expiry       string `json:"expiry"`        // The expiry date/time of the access token.
}

/*
TLS configuration structure to specify the settings for secure (TLS) connections.
This structure defines whether TLS is enabled, whether verification is required,
and the path to the certificate if TLS is enabled.
*/

// TLS represents the configuration for a TLS (Transport Layer Security) connection.
type TLS struct {
	Enabled         bool   // Whether TLS is enabled for the connection.
	Verification    bool   // Whether TLS certificate verification is enabled.
	CertificatePath string // The path to the TLS certificate file (if TLS is enabled).
}

/*
   Helper functions to generate default or custom configurations for credentials and TLS settings.
*/

// GetNoTLSConfiguration returns a default configuration with TLS disabled.
func GetNoTLSConfiguration() TLS {
	return TLS{
		Enabled:         false, // TLS is disabled by default.
		Verification:    false, // No certificate verification.
		CertificatePath: "",    // No certificate path needed.
	}
}

// GetTLSConfiguration returns a TLS configuration with the specified verification setting
// and certificate path for secure connections.
func GetTLSConfiguration(verification bool, certificatePath string) TLS {
	return TLS{
		Enabled:         true,            // TLS is enabled.
		Verification:    verification,    // Set TLS verification as specified.
		CertificatePath: certificatePath, // Path to the certificate if needed.
	}
}

// GetNoAccountCredentials returns a default account credentials structure with empty fields.
func GetNoAccountCredentials() AccountCredentials {
	return AccountCredentials{
		Username: "", // Empty username.
		Password: "", // Empty password.
	}
}

// GetAccountCredentials returns an AccountCredentials structure populated with the given
// username and password.
func GetAccountCredentials(username string, password string) AccountCredentials {
	return AccountCredentials{
		Username: username, // Provided username.
		Password: password, // Provided password.
	}
}
