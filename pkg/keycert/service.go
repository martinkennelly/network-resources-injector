package keycert

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/channel"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/service"
)

const (
	chBufferSize = 1
	serviceName = "key cert updater"
)

type keyCertUpdate struct {
	status  *channel.Channel
	quit    *channel.Channel
	timeout time.Duration
	keyCert Identity
	name string
}

// NewKeyCertUpdater offers functionality to monitor cert and key - changes to cert and key will trigger update of HTTP server
// cert and key.
func NewKeyCertUpdater(keyCert Identity, to time.Duration) service.Service {
	return &keyCertUpdate{nil, nil, to, keyCert, serviceName}
}

// Run checks if key & cert exist and start to monitor these files. Quit must be called after Run.
func (kcw *keyCertUpdate) Run() error {
	if kcw.status != nil && kcw.status.IsOpen() {
		return errors.New("key pair updater must have exited before attempting to run again")
	}
	kcw.status = channel.NewChannel(chBufferSize)
	kcw.quit = channel.NewChannel(chBufferSize)
	cert := kcw.keyCert.GetCertPath()
	key := kcw.keyCert.GetKeyPath()

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
func (kcw *keyCertUpdate) monitor() (err error) {
	defer func() {
		if err != nil {
			glog.Error(err)
		}
	}()
	glog.Info("starting TLS key and cert file updater")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	certUpdated := false
	keyUpdated := false
	watcher.Add(kcw.keyCert.GetCertPath())
	watcher.Add(kcw.keyCert.GetKeyPath())
	kcw.quit.Open()
	kcw.status.Open()
	defer kcw.status.Close()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				glog.Error("updater event received but not OK")
				continue
			}
			glog.Infof("updater event: '%v'", event)
			mask := fsnotify.Create | fsnotify.Rename | fsnotify.Remove |
				fsnotify.Write | fsnotify.Chmod
			if (event.Op & mask) != 0 {
				glog.Infof("modified file: '%v'", event.Name)
				if event.Name == kcw.keyCert.GetCertPath() {
					certUpdated = true
				}
				if event.Name == kcw.keyCert.GetKeyPath() {
					keyUpdated = true
				}
				if keyUpdated && certUpdated {
					if errReload := kcw.keyCert.Reload(); errReload != nil {
						err = fmt.Errorf("failed to reload certificate: '%v'", errReload)
						return
					}
					certUpdated = false
					keyUpdated = false
				}
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				glog.Errorf("updater error received but got error: '%s'", watchErr.Error())
				continue
			}
			err = fmt.Errorf("updater error: '%s'", watchErr)
			return
		case <-kcw.quit.GetCh():
			glog.Info("TLS cert and key file updater finished")
			return
		}
	}
}

// Quit attempts to terminate key/cert updater go routine and blocks until it ends. Quit call follows Run call. Error
// only when timeout occurs while waiting for updater to close
func (kcw *keyCertUpdate) Quit() error {
	glog.Info("terminating TLS cert & key updater")
	kcw.quit.Close()
	return kcw.status.WaitUntilClosed(kcw.timeout)
}

// StatusSignal returns channel that indicates when key/cert updater has ended. Channel will be closed if updater ends
func (kcw *keyCertUpdate) StatusSignal() chan struct{} {
	return kcw.status.GetCh()
}

// GetName returns service name
func (kcw *keyCertUpdate) GetName() string {
	return kcw.name
}