package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/golang/glog"
	nri "github.com/k8snetworkplumbingwg/network-resources-injector/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apiMError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const userDefinedInjectionConfigMap = "nri-user-defined-injections"

type udiService struct {
	status  *Channel
	quit    *Channel
	timeout time.Duration
	interval time.Duration
	namespace string
	udi     *userDefinedInjections
}

// package global needed due to http server calling a function (httpServerHandler) per request and unable to pass function args
var udiSvc *udiService

// GetUDI provides functionality to store user defined injections and updating this store periodically
func GetUDI(namespace string, interval, timeout time.Duration) nri.Service {
	if udiSvc == nil {
		udiSvc = &udiService{udi: &userDefinedInjections{Patchs: make(map[string]jsonPatchOperation)}, namespace: namespace,
			interval: interval, timeout: timeout}
	}
	return udiSvc
}

// Run will periodically update user defined injections from a config map. Quit must be called after Run.
func (udi *udiService) Run() error {
	if udi.status != nil && udi.status.IsOpen() {
		return errors.New("customized injection service must have exited before attempting to run again")
	}
	udi.status = NewChannel()
	udi.quit = NewChannel()

	go udi.monitorConfigMap()

	return udi.status.WaitUntilOpened(udi.timeout)
}

func (udi *udiService) monitorConfigMap() (err error) {
	glog.Info("starting user defined injection service")
	udi.quit.Open()
	udi.status.Open()
	defer udi.status.Close()
	ticker := time.NewTicker(udi.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm, err := clientset.CoreV1().ConfigMaps(udi.namespace).Get(
				context.Background(), userDefinedInjectionConfigMap, metav1.GetOptions{})
			if err != nil {
				if !apiMError.IsNotFound(err) {
					glog.Warningf("failed to get configmap for user-defined injections: %v", err)
					continue
				}
			}
			udi.setUserDefinedInjections(cm)
		case <-udi.quit.GetCh():
			glog.Info("user defined injection service finished")
			return
		}
	}
}

// Quit will stop updating the user define injections store
func (udi *udiService) Quit() error {
	glog.Info("terminating user defined injection service")
	udi.quit.Close()
	return udi.status.WaitUntilClosed(udi.timeout)
}

// StatusSignal returns a channel which indicates whether the user defined injections update service is running or not.
// channel closed indicates, it is not running.
func (udi *udiService) StatusSignal() chan struct{} {
	return udi.status.GetCh()
}


// setUserDefinedInjections sets additional injections to be applied in Pod spec
func (udi *udiService) setUserDefinedInjections(injections *corev1.ConfigMap) {
	// lock for writing
	udi.udi.Lock()
	defer udi.udi.Unlock()

	var patch jsonPatchOperation
	var userDefinedPatchs = udi.udi.Patchs

	for k, v := range injections.Data {
		existValue, exists := userDefinedPatchs[k]
		// unmarshall userDefined injection to json patch
		err := json.Unmarshal([]byte(v), &patch)
		if err != nil {
			glog.Errorf("failed to unmarshall user-defined injection: %v", v)
			continue
		}
		// metadata.Annotations is the only supported field for user definition
		// JsonPatchOperation.Path should be "/metadata/annotations"
		if patch.Path != "/metadata/annotations" {
			glog.Errorf("path: %v is not supported, only /metadata/annotations can be defined by user", patch.Path)
			continue
		}
		if !exists || !reflect.DeepEqual(existValue, patch) {
			glog.Infof("initializing user-defined injections with key: %v, value: %v", k, v)
			userDefinedPatchs[k] = patch
		}
	}
	// remove stale entries from userDefined configMap
	for k, _ := range userDefinedPatchs {
		if _, ok := injections.Data[k]; ok {
			continue
		}
		glog.Infof("removing stale entry: %v from user-defined injections", k)
		delete(userDefinedPatchs, k)
	}
}