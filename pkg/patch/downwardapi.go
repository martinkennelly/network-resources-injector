package patch

import (
	"strconv"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	annotationsPath        = "annotations"
	labelsPath             = "labels"
	downwardAPIMountPath   = "/etc/podnetinfo"
	hugepages1GRequestPath = "hugepages_1G_request"
	hugepages2MRequestPath = "hugepages_2M_request"
	hugepages1GLimitPath   = "hugepages_1G_limit"
	hugepages2MLimitPath   = "hugepages_2M_limit"
	envNameContainerName   = "CONTAINER_NAME"
)

type HugepageResourceData struct {
	ResourceName  string
	ContainerName string
	Path          string
}

func CreateHugepages(patch []JSONOperation, pod *v1.Pod) ([]JSONOperation, []HugepageResourceData) {
	// Determine if hugepages are being requested for a given container,
	// and if so, expose the value to the container via Downward API.
	var hugepageResourceList []HugepageResourceData

	for containerIndex, container := range pod.Spec.Containers {
		found := false
		if len(container.Resources.Requests) != 0 {
			if quantity, exists := container.Resources.Requests["hugepages-1Gi"]; exists && quantity.IsZero() == false {
				hugepageResource := HugepageResourceData{
					ResourceName:  "requests.hugepages-1Gi",
					ContainerName: container.Name,
					Path:          hugepages1GRequestPath + "_" + container.Name,
				}
				hugepageResourceList = append(hugepageResourceList, hugepageResource)
				found = true
			}
			if quantity, exists := container.Resources.Requests["hugepages-2Mi"]; exists && quantity.IsZero() == false {
				hugepageResource := HugepageResourceData{
					ResourceName:  "requests.hugepages-2Mi",
					ContainerName: container.Name,
					Path:          hugepages2MRequestPath + "_" + container.Name,
				}
				hugepageResourceList = append(hugepageResourceList, hugepageResource)
				found = true
			}
		}
		if len(container.Resources.Limits) != 0 {
			if quantity, exists := container.Resources.Limits["hugepages-1Gi"]; exists && quantity.IsZero() == false {
				hugepageResource := HugepageResourceData{
					ResourceName:  "limits.hugepages-1Gi",
					ContainerName: container.Name,
					Path:          hugepages1GLimitPath + "_" + container.Name,
				}
				hugepageResourceList = append(hugepageResourceList, hugepageResource)
				found = true
			}
			if quantity, exists := container.Resources.Limits["hugepages-2Mi"]; exists && quantity.IsZero() == false {
				hugepageResource := HugepageResourceData{
					ResourceName:  "limits.hugepages-2Mi",
					ContainerName: container.Name,
					Path:          hugepages2MLimitPath + "_" + container.Name,
				}
				hugepageResourceList = append(hugepageResourceList, hugepageResource)
				found = true
			}
		}

		// If Hugepages are being added to Downward API, add the
		// 'container.Name' as an environment variable to the container
		// so container knows its name and can process hugepages properly.
		if found {
			patch = createEnv(patch, &container, containerIndex, envNameContainerName, container.Name)
		}
	}
	return patch, hugepageResourceList
}

func CreateVol(patch []JSONOperation, hugepageResourceList []HugepageResourceData, pod *v1.Pod) []JSONOperation {
	patch = addVolumeMount(patch, len(pod.Spec.Containers))
	patch = addVolDownwardAPI(patch, hugepageResourceList, pod)
	return patch
}

func addVolumeMount(patch []JSONOperation, containersLen int) []JSONOperation {

	vm := v1.VolumeMount{
		Name:      "podnetinfo",
		ReadOnly:  true,
		MountPath: downwardAPIMountPath,
	}
	for containerIndex := 0; containerIndex < containersLen; containerIndex++ {
		patch = append(patch, JSONOperation{
			Operation: "add",
			Path:      "/spec/containers/" + strconv.Itoa(containerIndex) + "/volumeMounts/-",
			Value:     vm,
		})
	}
	return patch
}

func addVolDownwardAPI(patch []JSONOperation, hugepageResourceList []HugepageResourceData, pod *v1.Pod) []JSONOperation {
	var dAPIItems []v1.DownwardAPIVolumeFile

	if pod.Labels != nil && len(pod.Labels) > 0 {
		labels := v1.ObjectFieldSelector{
			FieldPath: "metadata.labels",
		}
		dAPILabels := v1.DownwardAPIVolumeFile{
			Path:     labelsPath,
			FieldRef: &labels,
		}
		dAPIItems = append(dAPIItems, dAPILabels)
	}

	if pod.Annotations != nil && len(pod.Annotations) > 0 {
		annotations := v1.ObjectFieldSelector{
			FieldPath: "metadata.annotations",
		}
		dAPIAnnotations := v1.DownwardAPIVolumeFile{
			Path:     annotationsPath,
			FieldRef: &annotations,
		}
		dAPIItems = append(dAPIItems, dAPIAnnotations)
	}

	for _, hugepageResource := range hugepageResourceList {
		hugepageSelector := v1.ResourceFieldSelector{
			Resource:      hugepageResource.ResourceName,
			ContainerName: hugepageResource.ContainerName,
			Divisor:       *resource.NewQuantity(1*1024*1024, resource.BinarySI),
		}
		dAPIHugepage := v1.DownwardAPIVolumeFile{
			Path:             hugepageResource.Path,
			ResourceFieldRef: &hugepageSelector,
		}
		dAPIItems = append(dAPIItems, dAPIHugepage)
	}

	dAPIVolSource := v1.DownwardAPIVolumeSource{
		Items: dAPIItems,
	}
	volSource := v1.VolumeSource{
		DownwardAPI: &dAPIVolSource,
	}
	vol := v1.Volume{
		Name:         "podnetinfo",
		VolumeSource: volSource,
	}

	patch = append(patch, JSONOperation{
		Operation: "add",
		Path:      "/spec/volumes/-",
		Value:     vol,
	})
	return patch
}
