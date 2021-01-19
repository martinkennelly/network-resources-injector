package auth

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
)

type ClientCAPool interface {
	Load() error
	GetCertPool() *x509.CertPool
}

type clientCertPool struct {
	certPool  *x509.CertPool
	certPaths ClientCAFlags
	insecure  bool
}

// NewClientCertPool will load a single client CA
func NewClientCertPool(clientCaPaths ClientCAFlags, insecure bool) (ClientCAPool, error) {
	pool := &clientCertPool{
		certPaths: clientCaPaths,
		insecure:  insecure,
	}
	if !pool.insecure {
		if err := pool.Load(); err != nil {
			return nil, err
		}
	}
	return pool, nil
}

// Load a certificate into the client CA pool
func (pool *clientCertPool) Load() error {
	if pool.insecure {
		glog.Infof("can not load client CA pool. Remove --insecure flag to enable.")
		return nil
	}

	if len(pool.certPaths) == 0 {
		return fmt.Errorf("no client CA file path(s) found")
	}

	pool.certPool = x509.NewCertPool()
	for _, path := range pool.certPaths {
		caCertPem, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load client CA file from path '%s'", path)
		}
		if ok := pool.certPool.AppendCertsFromPEM(caCertPem); !ok {
			return fmt.Errorf("failed to parse client CA file from path '%s'", path)
		}
		glog.Infof("added client CA to cert pool from path '%s'", path)
	}
	glog.Infof("added '%d' client CA(s) to cert pool", len(pool.certPaths))
	return nil
}

// GetCertPool returns a client CA pool
func (pool *clientCertPool) GetCertPool() *x509.CertPool {
	if pool.insecure {
		return nil
	}
	return pool.certPool
}
