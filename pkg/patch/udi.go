package patch

import (
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
)

type UserDefinedInjections struct {
	sync.Mutex
	Patches map[string]JSONOperation
}

func (udi *UserDefinedInjections) CreateCustomizedPatch(pod v1.Pod) ([]JSONOperation, error) {
	var userDefinedPatch []JSONOperation

	// lock for reading
	udi.Lock()
	defer udi.Unlock()

	for k, v := range udi.Patches {
		// The userDefinedInjects will be injected when:
		// 1. Pod labels contain the patch key defined in userDefinedInjects, and
		// 2. The value of patch key in pod labels(not in userDefinedInjects) is "true"
		if podValue, exists := pod.ObjectMeta.Labels[k]; exists && strings.ToLower(podValue) == "true" {
			userDefinedPatch = append(userDefinedPatch, v)
		}
	}
	return userDefinedPatch, nil
}
