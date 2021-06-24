package keycert

import (
	"crypto/tls"
	"sync"

	"github.com/golang/glog"
)

type Identity interface {
	Reload() error
	GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	GetKeyPath() string
	GetCertPath() string
}

type keyCert struct {
	certMutex sync.RWMutex
	cert      *tls.Certificate
	certPath  string
	keyPath   string
}

// NewIdentity loads a cert and key
func NewIdentity(certPath, keyPath string) (Identity, error) {
	result := &keyCert{
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

func (keyPair *keyCert) Reload() error {
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

func (keyPair *keyCert) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		keyPair.certMutex.RLock()
		defer keyPair.certMutex.RUnlock()
		return keyPair.cert, nil
	}
}

func (keyPair *keyCert) GetKeyPath() string {
	return keyPair.keyPath
}

func (keyPair *keyCert) GetCertPath() string {
	return keyPair.certPath
}
