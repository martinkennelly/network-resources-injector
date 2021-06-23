package tls

import (
	"crypto/tls"
	"sync"

	"github.com/golang/glog"
)

type KeyReloader interface {
	Reload() error
	GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	GetKeyPath() string
	GetCertPath() string
}

type tlsKeypairReloader struct {
	certMutex sync.RWMutex
	cert      *tls.Certificate
	certPath  string
	keyPath   string
}

//NewTlsKeyPairReloader loads a cert and key
func NewTlsKeyPairReloader(certPath, keyPath string) (KeyReloader, error) {
	result := &tlsKeypairReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	result.cert = &cert

	return result, nil
}

func (keyPair *tlsKeypairReloader) Reload() error {
	newCert, err := tls.LoadX509KeyPair(keyPair.certPath, keyPair.keyPath)
	if err != nil {
		return err
	}
	glog.V(2).Infof("certificate reloaded")
	keyPair.certMutex.Lock()
	defer keyPair.certMutex.Unlock()
	keyPair.cert = &newCert
	return nil
}

func (keyPair *tlsKeypairReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		keyPair.certMutex.RLock()
		defer keyPair.certMutex.RUnlock()
		return keyPair.cert, nil
	}
}

func (keyPair *tlsKeypairReloader) GetKeyPath() string {
	return keyPair.keyPath
}

func (keyPair *tlsKeypairReloader) GetCertPath() string {
	return keyPair.certPath
}
