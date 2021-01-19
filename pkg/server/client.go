package server

import "crypto/tls"

// getClientAuth determines the policy the http server will follow for TLS Client Authentication
func getClientAuth(insecure bool) tls.ClientAuthType {
	if insecure {
		return tls.NoClientCert
	}
	return tls.RequireAndVerifyClientCert
}
