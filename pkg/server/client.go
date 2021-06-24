package server

import "crypto/tls"

// GetClientAuth determines the policy the http server will follow for TLS Client Authentication
func GetClientAuth(insecure bool) tls.ClientAuthType {
	if insecure {
		return tls.NoClientCert
	}
	return tls.RequireAndVerifyClientCert
}
