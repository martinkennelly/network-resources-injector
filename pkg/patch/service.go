package patch

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	apiMError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/channel"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/client"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/service"
)

const (
	userDefinedInjectionConfigMap = "nri-user-defined-injections"
	serviceName                   = "user defined injections updater"
	chBufferSize                  = 1
)

type udiUpdate struct {
	status    *channel.Channel
	quit      *channel.Channel
	timeout   time.Duration
	interval  time.Duration
	namespace string
	name      string
	client    kubernetes.Interface
	udi       *UserDefinedInjections
}

// package global needed due to HTTP server calling a function (httpServerHandler) per request and unable to pass this var
// via function args
var udiSvc *udiUpdate

// CreateUDIUpdater provides functionality to store user defined injections and updating this store periodically
func CreateUDIUpdater(namespace string, interval, timeout time.Duration) service.Service {
	udiSvc = &udiUpdate{timeout: timeout, interval: interval,  namespace: namespace, name: serviceName,
		udi: &UserDefinedInjections{Mutex: sync.Mutex{}, Patches: make(map[string]JSONOperation)}}

	return udiSvc
}

// GetUDI return user defined injections
func GetUDI() *UserDefinedInjections {
	if udiSvc != nil {
		return udiSvc.udi
	}
	return nil
}

// Run will periodically update user defined injections from a config map. Quit must be called after Run.
func (udiU *udiUpdate) Run() error {
	if udiU.status != nil && udiU.status.IsOpen() {
		return fmt.Errorf(fmt.Sprintf("%s must have exited before attempting to run again", udiU.name))
	}
	udiU.status = channel.NewChannel(chBufferSize)
	udiU.quit = channel.NewChannel(chBufferSize)

	if udiU.client == nil {
		k8Client, err := client.GetInClusterClient()
		if err != nil {
			return err
		}
		udiU.client = k8Client
	}

	go udiU.monitorConfigMap()

	return udiU.status.WaitUntilOpened(udiU.timeout)
}

func (udiU *udiUpdate) monitorConfigMap() {
	glog.Info(fmt.Sprintf("starting %s", udiU.name))
	udiU.quit.Open()
	udiU.status.Open()
	defer udiU.status.Close()
	ticker := time.NewTicker(udiU.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm, err := udiU.client.CoreV1().ConfigMaps(udiU.namespace).Get(
				context.Background(), userDefinedInjectionConfigMap, metav1.GetOptions{})
			if err != nil {
				if !apiMError.IsNotFound(err) {
					glog.Warningf("failed to get configmap for user-defined injections: %v", err)
					continue
				}
			}
			udiU.setUserDefinedInjections(cm)
		case <-udiU.quit.GetCh():
			glog.Info(fmt.Sprintf("%s finished", udiU.name))
			return
		}
	}
}

// Quit will stop updating the user define injections store
func (udiU *udiUpdate) Quit() error {
	glog.Info(fmt.Sprintf("terminating %s", udiU.name))
	udiU.quit.Close()
	return udiU.status.WaitUntilClosed(udiU.timeout)
}

// StatusSignal returns a channel which indicates whether the user defined injections update service is running or not.
// channel closed indicates, it is not running.
func (udiU *udiUpdate) StatusSignal() chan struct{} {
	return udiU.status.GetCh()
}

// GetName return service name
func (udiU *udiUpdate) GetName() string {
	return udiU.name
}

// setUserDefinedInjections sets additional injections to be applied in Pod spec
func (udiU *udiUpdate) setUserDefinedInjections(injections *corev1.ConfigMap) {
	// lock for writing
	udiU.udi.Lock()
	defer udiU.udi.Unlock()

	var patch JSONOperation
	var userDefinedPatches = udiU.udi.Patches

	for k, v := range injections.Data {
		existValue, exists := userDefinedPatches[k]
		// unmarshall userDefined injection to json patch
		err := json.Unmarshal([]byte(v), &patch)
		if err != nil {
			glog.Errorf("failed to unmarshall user-defined injection: %v", v)
			continue
		}
		// metadata.Annotations is the only supported field for user definition
		// JSONOperation.Path should be "/metadata/annotations"
		if patch.Path != "/metadata/annotations" {
			glog.Errorf("path: %v is not supported, only /metadata/annotations can be defined by user", patch.Path)
			continue
		}
		if !exists || !reflect.DeepEqual(existValue, patch) {
			glog.Infof("initializing user-defined injections with key: %v, value: %v", k, v)
			userDefinedPatches[k] = patch
		}
	}
	// remove stale entries from userDefined configMap
	for k := range userDefinedPatches {
		if _, ok := injections.Data[k]; ok {
			continue
		}
		glog.Infof("removing stale entry: %v from user-defined injections", k)
		delete(userDefinedPatches, k)
	}
}
