package webhook

import (
	"errors"
	"fmt"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/channel"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/service"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/tls"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
)

type keyPairService struct {
	status  *channel.Channel
	quit    *channel.Channel
	timeout time.Duration
	keyPair tls.KeyReloader
}

// NewKeyPair offers functionality to monitor cert and key - changes to cert and key will trigger update of HTTP server
// cert and key.
func NewKeyPair(keyCert tls.KeyReloader, to time.Duration) service.Service {
	return &keyPairService{nil, nil, to, keyCert}
}

// Run checks if key & cert exist and start to monitor these files. Quit must be called after Run.
func (kcw *keyPairService) Run() error {
	if kcw.status != nil && kcw.status.IsOpen() {
		return errors.New("key pair watcher must have exited before attempting to run again")
	}
	kcw.status = channel.NewChannel()
	kcw.quit = channel.NewChannel()
	cert := kcw.keyPair.GetCertPath()
	key := kcw.keyPair.GetKeyPath()

	if cert == "" || key == "" {
		return errors.New("cert and/or key path are not set")
	}
	if _, errStat := os.Stat(cert); os.IsNotExist(errStat) {
		return fmt.Errorf("cert file does not exist at path '%s'", cert)
	}
	if _, errStat := os.Stat(key); os.IsNotExist(errStat) {
		return fmt.Errorf("key file does not exist at path '%s'", key)
	}

	go kcw.monitor()

	return kcw.status.WaitUntilOpened(kcw.timeout)
}

// monitor key & cert files. Finish when quit signal received
func (kcw *keyPairService) monitor() (err error) {
	defer func() {
		if err != nil {
			glog.Error(err)
		}
	}()
	glog.Info("starting TLS key and cert file watcher")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	certUpdated := false
	keyUpdated := false
	watcher.Add(kcw.keyPair.GetCertPath())
	watcher.Add(kcw.keyPair.GetKeyPath())
	kcw.quit.Open()
	kcw.status.Open()
	defer kcw.status.Close()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				glog.Error("watcher event received but not OK")
				continue
			}
			glog.Infof("watcher event: '%v'", event)
			mask := fsnotify.Create | fsnotify.Rename | fsnotify.Remove |
				fsnotify.Write | fsnotify.Chmod
			if (event.Op & mask) != 0 {
				glog.Infof("modified file: '%v'", event.Name)
				if event.Name == kcw.keyPair.GetCertPath() {
					certUpdated = true
				}
				if event.Name == kcw.keyPair.GetKeyPath() {
					keyUpdated = true
				}
				if keyUpdated && certUpdated {
					if errReload := kcw.keyPair.Reload(); errReload != nil {
						err = fmt.Errorf("failed to reload certificate: '%v'", errReload)
						return
					}
					certUpdated = false
					keyUpdated = false
				}
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				glog.Errorf("watcher error received but got error: '%s'", watchErr.Error())
				continue
			}
			err = fmt.Errorf("watcher error: '%s'", watchErr)
			return
		case <-kcw.quit.GetCh():
			glog.Info("TLS cert and key file watcher finished")
			return
		}
	}
}

// Quit attempts to terminate key/cert watcher go routine and blocks until it ends. Quit call follows Run call. Error
// only when timeout occurs while waiting for watcher to close
func (kcw *keyPairService) Quit() error {
	glog.Info("terminating TLS cert & key watcher")
	kcw.quit.Close()
	return kcw.status.WaitUntilClosed(kcw.timeout)
}

// StatusSignal returns channel that indicates when key/cert watcher has ended. Channel will be closed if watcher ends
func (kcw *keyPairService) StatusSignal() chan struct{} {
	return kcw.status.GetCh()
}