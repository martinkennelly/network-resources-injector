package tls

import "crypto/tls"

type ClientCAFlags []string

func (i *ClientCAFlags) String() string {
	return ""
}

func (i *ClientCAFlags) Set(path string) error {
	*i = append(*i, path)
	return nil
}

//GetClientAuth determines the policy the http server will follow for TLS Client Authentication
func GetClientAuth(insecure bool) tls.ClientAuthType {
	if insecure {
		return tls.NoClientCert
	}
	return tls.RequireAndVerifyClientCert
}
