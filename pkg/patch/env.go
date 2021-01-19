package patch

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

func createEnv(patch []JSONOperation, container *v1.Container, containerIndex int, envName string,
	envVal string) []JSONOperation {

	// Determine if requested ENV already exists
	found := false
	firstElement := false
	if len(container.Env) != 0 {
		for _, env := range container.Env {
			if env.Name == envName {
				found = true
				if env.Value != envVal {
					glog.Warningf("Error, adding env '%s', name existed but value different: '%s' != '%s'",
						envName, env.Value, envVal)
				}
				break
			}
		}
	} else {
		firstElement = true
	}

	if !found {
		patch = addEnvVar(patch, containerIndex, firstElement, envName, envVal)
	}
	return patch
}

func addEnvVar(patch []JSONOperation, containerIndex int, firstElement bool, envName string, envVal string) []JSONOperation {
	env := v1.EnvVar{
		Name:  envName,
		Value: envVal,
	}

	if firstElement {
		patch = append(patch, JSONOperation{
			Operation: "add",
			Path:      "/spec/containers/" + fmt.Sprintf("%d", containerIndex) + "/env",
			Value:     []v1.EnvVar{env},
		})
	} else {
		patch = append(patch, JSONOperation{
			Operation: "add",
			Path:      "/spec/containers/" + fmt.Sprintf("%d", containerIndex) + "/env/-",
			Value:     env,
		})
	}
	return patch
}
