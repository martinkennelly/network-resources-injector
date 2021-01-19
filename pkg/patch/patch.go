package patch

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

type JSONOperation struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

func AppendCustomized(patch []JSONOperation, pod *v1.Pod, userDefinedPatch []JSONOperation) []JSONOperation {
	//Add operation for annotations is currently only supported
	return appendAddAnnot(patch, pod, userDefinedPatch)
}

func appendAddAnnot(patch []JSONOperation, pod *v1.Pod, userDefinedPatch []JSONOperation) []JSONOperation {
	annotations := make(map[string]string)
	patchOp := JSONOperation{
		Operation: "add",
		Path:      "/metadata/annotations",
		Value:     annotations,
	}

	for _, p := range userDefinedPatch {
		if p.Path == "/metadata/annotations" && p.Operation == "add" {
			//loop over user defined injected annotations key-value pairs
			for k, v := range p.Value.(map[string]interface{}) {
				if _, exists := annotations[k]; exists {
					glog.Warningf("ignoring duplicate user defined injected annotation: %s: %s", k, v.(string))
				} else {
					annotations[k] = v.(string)
				}
			}
		}
	}

	if len(annotations) > 0 {
		// attempt to add existing pod annotation but do not override
		for k, v := range pod.ObjectMeta.Annotations {
			if _, exists := annotations[k]; !exists {
				annotations[k] = v
			}
		}
		patch = append(patch, patchOp)
	}
	return patch
}
