package patch

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

func GetNetworkSelections(annotationKey string, pod v1.Pod, userDefinedPatch []JSONOperation) (string, bool) {
	// User defined annotateKey takes precedence than userDefined injections
	glog.Infof("search %s in original pod annotations", annotationKey)
	nets, exists := pod.ObjectMeta.Annotations[annotationKey]
	if exists {
		glog.Infof("%s is defined in original pod annotations", annotationKey)
		return nets, exists
	}

	glog.Infof("search %s in user-defined injections", annotationKey)
	// userDefinedPatch may contain user defined net-attach-defs
	if len(userDefinedPatch) > 0 {
		for _, p := range userDefinedPatch {
			if p.Operation == "add" && p.Path == "/metadata/annotations" {
				for k, v := range p.Value.(map[string]interface{}) {
					if k == annotationKey {
						glog.Infof("%s is found in user-defined annotations", annotationKey)
						return v.(string), true
					}
				}
			}
		}
	}
	glog.Infof("%s is not found in either pod annotations or user-defined injections", annotationKey)
	return "", false
}
