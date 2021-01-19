package patch

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func CreateResource(patch []JSONOperation, Containers []v1.Container, resourceRequests map[string]int64) []JSONOperation {
	/* check whether resources paths exists in the first container and add as the first patches if missing */
	if len(Containers[0].Resources.Requests) == 0 {
		patch = patchEmptyResources(patch, 0, "requests")
	}
	if len(Containers[0].Resources.Limits) == 0 {
		patch = patchEmptyResources(patch, 0, "limits")
	}

	for resourceName := range resourceRequests {
		for _, container := range Containers {
			if _, exists := container.Resources.Limits[v1.ResourceName(resourceName)]; exists {
				delete(resourceRequests, resourceName)
			}
			if _, exists := container.Resources.Requests[v1.ResourceName(resourceName)]; exists {
				delete(resourceRequests, resourceName)
			}
		}
	}

	resourceList := *getResourceList(resourceRequests)

	for res, quantity := range resourceList {
		patch = appendResource(patch, res.String(), quantity, quantity)
	}

	return patch
}

func UpdateResource(patch []JSONOperation, Containers []v1.Container, resourceRequests map[string]int64) []JSONOperation {
	var existingrequestsMap map[v1.ResourceName]resource.Quantity
	var existingLimitsMap map[v1.ResourceName]resource.Quantity

	if len(Containers[0].Resources.Requests) == 0 {
		patch = patchEmptyResources(patch, 0, "requests")
	} else {
		existingrequestsMap = Containers[0].Resources.Requests
	}
	if len(Containers[0].Resources.Limits) == 0 {
		patch = patchEmptyResources(patch, 0, "limits")
	} else {
		existingLimitsMap = Containers[0].Resources.Limits
	}

	resourceList := *getResourceList(resourceRequests)

	for resourceName, quantity := range resourceList {
		reqQuantity := quantity
		limitQuantity := quantity
		if value, ok := existingrequestsMap[resourceName]; ok {
			reqQuantity.Add(value)
		}
		if value, ok := existingLimitsMap[resourceName]; ok {
			limitQuantity.Add(value)
		}
		patch = appendResource(patch, resourceName.String(), reqQuantity, limitQuantity)
	}

	return patch
}

func appendResource(patch []JSONOperation, resourceName string, reqQuantity, limitQuantity resource.Quantity) []JSONOperation {
	patch = append(patch, JSONOperation{
		Operation: "add",
		Path:      "/spec/containers/0/resources/requests/" + toSafeJsonPatchKey(resourceName),
		Value:     reqQuantity,
	})
	patch = append(patch, JSONOperation{
		Operation: "add",
		Path:      "/spec/containers/0/resources/limits/" + toSafeJsonPatchKey(resourceName),
		Value:     limitQuantity,
	})

	return patch
}

func getResourceList(resourceRequests map[string]int64) *v1.ResourceList {
	resourceList := v1.ResourceList{}
	for name, number := range resourceRequests {
		resourceList[v1.ResourceName(name)] = *resource.NewQuantity(number, resource.DecimalSI)
	}

	return &resourceList
}

func patchEmptyResources(patch []JSONOperation, containerIndex uint, key string) []JSONOperation {
	patch = append(patch, JSONOperation{
		Operation: "add",
		Path:      "/spec/containers/" + fmt.Sprintf("%d", containerIndex) + "/resources/" + toSafeJsonPatchKey(key),
		Value:     v1.ResourceList{},
	})
	return patch
}

func toSafeJsonPatchKey(in string) string {
	out := strings.Replace(in, "~", "~0", -1)
	out = strings.Replace(out, "/", "~1", -1)
	return out
}
