package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang/glog"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/pkg/errors"
	multus "gopkg.in/intel/multus-cni.v3/pkg/types"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"

	netcache "github.com/k8snetworkplumbingwg/network-resources-injector/pkg/cache"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/client"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/patch"
)

const (
	networksAnnotationKey       = "k8s.v1.cni.cncf.io/networks"
	nodeSelectorKey             = "k8s.v1.cni.cncf.io/nodeSelector"
	defaultNetworkAnnotationKey = "v1.multus-cni.io/default-network"
)

var (
	injectHugepageDownApi  bool
	resourceNameKeys       []string
	honorExistingResources bool
)

// mutateHandler handles AdmissionReview requests and sends responses back to the K8s API server
func mutateHandler(w http.ResponseWriter, req *http.Request) {
	glog.Infof("Received mutation request")
	var err error

	/* read AdmissionReview from the HTTP request */
	ar, httpStatus, err := readAdmissionReview(req, w)
	if err != nil {
		http.Error(w, err.Error(), httpStatus)
		return
	}

	k8Client, err := client.GetInClusterClient()
	if err != nil {
		http.Error(w, "failed to get in-cluster config", http.StatusInternalServerError)
	}

	/* read pod annotations */
	/* if networks missing skip everything */
	pod, err := deserializePod(k8Client, ar)
	if err != nil {
		handleValidationError(w, ar, err)
		return
	}
	glog.Infof("AdmissionReview request received for pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	udiUpdater := patch.GetUDI()
	if udiUpdater == nil {
		http.Error(w, "user defined injection service not set", http.StatusInternalServerError)
		return
	}

	userDefinedPatch, err := udiUpdater.CreateCustomizedPatch(pod)
	if err != nil {
		glog.Warningf("failed to create user-defined injection patch for pod %s/%s, err: %v",
			pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
	}

	defaultNetSelection, defExist := patch.GetNetworkSelections(defaultNetworkAnnotationKey, pod, userDefinedPatch)
	additionalNetSelections, addExists := patch.GetNetworkSelections(networksAnnotationKey, pod, userDefinedPatch)

	if defExist || addExists {
		/* map of resources request needed by a pod and a number of them */
		resourceRequests := make(map[string]int64)

		/* map of node labels on which pod needs to be scheduled*/
		desiredNsMap := make(map[string]string)

		if defaultNetSelection != "" {
			defNetwork, err := parsePodNetworkSelections(defaultNetSelection, pod.ObjectMeta.Namespace)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if len(defNetwork) == 1 {
				resourceRequests, desiredNsMap, err = parseNetworkAttachDefinition(k8Client, defNetwork[0], resourceRequests, desiredNsMap)
				if err != nil {
					err = prepareAdmissionReviewResponse(false, err.Error(), ar)
					if err != nil {
						glog.Errorf("error preparing AdmissionReview response for pod %s/%s, error: %v",
							pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					writeResponse(w, ar)
					return
				}
			}
		}
		if additionalNetSelections != "" {
			/* unmarshal list of network selection objects */
			networks, err := parsePodNetworkSelections(additionalNetSelections, pod.ObjectMeta.Namespace)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, n := range networks {
				resourceRequests, desiredNsMap, err = parseNetworkAttachDefinition(k8Client, n, resourceRequests, desiredNsMap)
				if err != nil {
					err = prepareAdmissionReviewResponse(false, err.Error(), ar)
					if err != nil {
						glog.Errorf("error preparing AdmissionReview response for pod %s/%s, error: %v",
							pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					writeResponse(w, ar)
					return
				}
			}
			glog.Infof("pod %s/%s has resource requests: %v and node selectors: %v", pod.ObjectMeta.Namespace,
				pod.ObjectMeta.Name, resourceRequests, desiredNsMap)
		}

		/* patch with custom resources requests and limits */
		err = prepareAdmissionReviewResponse(true, "allowed", ar)
		if err != nil {
			glog.Errorf("error preparing AdmissionReview response for pod %s/%s, error: %v",
				pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var patches []patch.JSONOperation
		if len(resourceRequests) == 0 {
			glog.Infof("pod %s/%s doesn't need any custom network resources", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		} else {
			glog.Infof("honor-resources=%v", honorExistingResources)
			if honorExistingResources {
				patches = patch.UpdateResource(patches, pod.Spec.Containers, resourceRequests)
			} else {
				patches = patch.CreateResource(patches, pod.Spec.Containers, resourceRequests)
			}
			glog.Infof("injectHugepageDownApi=%v", injectHugepageDownApi)
			var hugepageResources []patch.HugepageResourceData
			if injectHugepageDownApi {
				patches, hugepageResources = patch.CreateHugepages(patches, &pod)
			}
			patches = patch.CreateVol(patches, hugepageResources, &pod)
			patches = patch.AppendCustomized(patches, &pod, userDefinedPatch)
		}
		patches = patch.CreateNodeSelector(patches, pod.Spec.NodeSelector, desiredNsMap)
		glog.Infof("patch after all mutations: %v for pod %s/%s", patches, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

		patchBytes, _ := json.Marshal(patches)
		ar.Response.Patch = patchBytes
		ar.Response.PatchType = func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}()
	} else {
		/* network annotation not provided or empty */
		glog.Infof("pod %s/%s spec doesn't have network annotations. Skipping...", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		err = prepareAdmissionReviewResponse(true, "Pod spec doesn't have network annotations. Skipping...", ar)
		if err != nil {
			glog.Errorf("error preparing AdmissionReview response for pod %s/%s, error: %v",
				pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	writeResponse(w, ar)
}

// SetResourceNameKeys extracts resources from a string and add them to resourceNameKeys array
func SetResourceNameKeys(keys string) error {
	if keys == "" {
		return errors.New("resoure keys can not be empty")
	}
	for _, resourceNameKey := range strings.Split(keys, ",") {
		resourceNameKey = strings.TrimSpace(resourceNameKey)
		resourceNameKeys = append(resourceNameKeys, resourceNameKey)
	}
	return nil
}

// SetInjectHugepageDownApi sets a flag to indicate whether or not to inject the
// hugepage request and limit for the Downward API.
func SetInjectHugepageDownApi(hugepageFlag bool) {
	injectHugepageDownApi = hugepageFlag
}

// SetHonorExistingResources initialize the honorExistingResources flag
func SetHonorExistingResources(resourcesHonorFlag bool) {
	honorExistingResources = resourcesHonorFlag
}

// httpServerHandler limits HTTP server endpoint to /mutate and HTTP verb to POST only
func httpServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != endpoint {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid HTTP verb requested", 405)
		return
	}
	mutateHandler(w, r)
}

func prepareAdmissionReviewResponse(allowed bool, message string, ar *v1beta1.AdmissionReview) error {
	if ar.Request != nil {
		ar.Response = &v1beta1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: allowed,
		}
		if message != "" {
			ar.Response.Result = &v1.Status{
				Message: message,
			}
		}
		return nil
	}
	return errors.New("received empty AdmissionReview request")
}

func readAdmissionReview(req *http.Request, w http.ResponseWriter) (*v1beta1.AdmissionReview, int, error) {
	var body []byte

	if req.Body != nil {
		req.Body = http.MaxBytesReader(w, req.Body, 1<<20)
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		err := errors.New("Error reading HTTP request: empty body")
		glog.Errorf("%s", err)
		return nil, http.StatusBadRequest, err
	}

	/* validate HTTP request headers */
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := errors.Errorf("Invalid Content-Type='%s', expected 'application/json'", contentType)
		glog.Errorf("%v", err)
		return nil, http.StatusUnsupportedMediaType, err
	}

	/* read AdmissionReview from the request body */
	ar, err := deserializeAdmissionReview(body)
	if err != nil {
		err := errors.Wrap(err, "error deserializing AdmissionReview")
		glog.Errorf("%v", err)
		return nil, http.StatusBadRequest, err
	}

	return ar, http.StatusOK, nil
}

func deserializeAdmissionReview(body []byte) (*v1beta1.AdmissionReview, error) {
	ar := &v1beta1.AdmissionReview{}
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)
	deserializer := codecs.UniversalDeserializer()
	_, _, err := deserializer.Decode(body, nil, ar)

	/* Decode() won't return an error if the data wasn't actual AdmissionReview */
	if err == nil && ar.TypeMeta.Kind != "AdmissionReview" {
		err = errors.New("received object is not an AdmissionReview")
	}

	return ar, err
}

func deserializePod(client kubernetes.Interface, ar *v1beta1.AdmissionReview) (corev1.Pod, error) {
	/* unmarshal Pod from AdmissionReview request */
	pod := corev1.Pod{}
	err := json.Unmarshal(ar.Request.Object.Raw, &pod)
	if pod.ObjectMeta.Namespace != "" {
		return pod, err
	}

	// AdmissionRequest contains an optional Namespace field
	if ar.Request.Namespace != "" {
		pod.ObjectMeta.Namespace = ar.Request.Namespace
		return pod, nil
	}

	ownerRef := pod.ObjectMeta.OwnerReferences
	if len(ownerRef) > 0 {
		namespace, err := getNamespaceFromOwnerReference(client, pod.ObjectMeta.OwnerReferences[0])
		if err != nil {
			return pod, err
		}
		pod.ObjectMeta.Namespace = namespace
	}

	// pod.ObjectMeta.Namespace may still be empty at this point,
	// but there is a chance that net-attach-def annotation contains
	// a valid namespace
	return pod, err
}

func getNamespaceFromOwnerReference(client kubernetes.Interface, ownerRef v1.OwnerReference) (namespace string, err error) {
	namespace = ""
	switch ownerRef.Kind {
	case "ReplicaSet":
		var replicaSets *appsv1.ReplicaSetList
		replicaSets, err = client.AppsV1().ReplicaSets("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return
		}
		for _, replicaSet := range replicaSets.Items {
			if replicaSet.ObjectMeta.Name == ownerRef.Name && replicaSet.ObjectMeta.UID == ownerRef.UID {
				namespace = replicaSet.ObjectMeta.Namespace
				err = nil
				break
			}
		}
	case "DaemonSet":
		var daemonSets *appsv1.DaemonSetList
		daemonSets, err = client.AppsV1().DaemonSets("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return
		}
		for _, daemonSet := range daemonSets.Items {
			if daemonSet.ObjectMeta.Name == ownerRef.Name && daemonSet.ObjectMeta.UID == ownerRef.UID {
				namespace = daemonSet.ObjectMeta.Namespace
				err = nil
				break
			}
		}
	case "StatefulSet":
		var statefulSets *appsv1.StatefulSetList
		statefulSets, err = client.AppsV1().StatefulSets("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return
		}
		for _, statefulSet := range statefulSets.Items {
			if statefulSet.ObjectMeta.Name == ownerRef.Name && statefulSet.ObjectMeta.UID == ownerRef.UID {
				namespace = statefulSet.ObjectMeta.Namespace
				err = nil
				break
			}
		}
	case "ReplicationController":
		var replicationControllers *corev1.ReplicationControllerList
		replicationControllers, err = client.CoreV1().ReplicationControllers("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return
		}
		for _, replicationController := range replicationControllers.Items {
			if replicationController.ObjectMeta.Name == ownerRef.Name && replicationController.ObjectMeta.UID == ownerRef.UID {
				namespace = replicationController.ObjectMeta.Namespace
				err = nil
				break
			}
		}
	default:
		glog.Infof("owner reference kind is not supported: %v, using default namespace", ownerRef.Kind)
		namespace = "default"
		return
	}

	if namespace == "" {
		err = errors.New("pod namespace is not found")
	}

	return

}

func parsePodNetworkSelections(podNetworks, defaultNamespace string) ([]*multus.NetworkSelectionElement, error) {
	var networkSelections []*multus.NetworkSelectionElement

	if len(podNetworks) == 0 {
		err := errors.New("empty string passed as network selection elements list")
		glog.Error(err)
		return nil, err
	}

	/* try to parse as JSON array */
	err := json.Unmarshal([]byte(podNetworks), &networkSelections)

	/* if failed, try to parse as comma separated */
	if err != nil {
		glog.Infof("'%s' is not in JSON format: %s... trying to parse as comma separated network selections list", podNetworks, err)
		for _, networkSelection := range strings.Split(podNetworks, ",") {
			networkSelection = strings.TrimSpace(networkSelection)
			networkSelectionElement, err := parsePodNetworkSelectionElement(networkSelection, defaultNamespace)
			if err != nil {
				err := errors.Wrap(err, "error parsing network selection element")
				glog.Error(err)
				return nil, err
			}
			networkSelections = append(networkSelections, networkSelectionElement)
		}
	}

	/* fill missing namespaces with default value */
	for _, networkSelection := range networkSelections {
		if networkSelection.Namespace == "" {
			if defaultNamespace == "" {
				// Ignore the AdmissionReview request when the following conditions are met:
				// 1) net-attach-def annotation doesn't contain a valid namespace
				// 2) defaultNamespace retrieved from admission request is empty
				// Pod admission would fail in subsquent call "getNetworkAttachmentDefinition"
				// if no namespace is specified. We don't want to fail the pod creation
				// in such case since it is possible that pod is not a SR-IOV pod
				glog.Warningf("The admission request doesn't contain a valid namespace, ignoring...")
				return nil, nil
			} else {
				networkSelection.Namespace = defaultNamespace
			}
		}
	}

	return networkSelections, nil
}

func parsePodNetworkSelectionElement(selection, defaultNamespace string) (*multus.NetworkSelectionElement, error) {
	var namespace, name, netInterface string
	var networkSelectionElement *multus.NetworkSelectionElement

	units := strings.Split(selection, "/")
	switch len(units) {
	case 1:
		namespace = defaultNamespace
		name = units[0]
	case 2:
		namespace = units[0]
		name = units[1]
	default:
		err := errors.Errorf("invalid network selection element - more than one '/' rune in: '%s'", selection)
		glog.Info(err)
		return networkSelectionElement, err
	}

	units = strings.Split(name, "@")
	switch len(units) {
	case 1:
		name = units[0]
		netInterface = ""
	case 2:
		name = units[0]
		netInterface = units[1]
	default:
		err := errors.Errorf("invalid network selection element - more than one '@' rune in: '%s'", selection)
		glog.Info(err)
		return networkSelectionElement, err
	}

	validNameRegex, _ := regexp.Compile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	for _, unit := range []string{namespace, name, netInterface} {
		ok := validNameRegex.MatchString(unit)
		if !ok && len(unit) > 0 {
			err := errors.Errorf("at least one of the network selection units is invalid: error found at '%s'", unit)
			glog.Info(err)
			return networkSelectionElement, err
		}
	}

	networkSelectionElement = &multus.NetworkSelectionElement{
		Namespace:        namespace,
		Name:             name,
		InterfaceRequest: netInterface,
	}

	return networkSelectionElement, nil
}

func getNetworkAttachmentDefinition(client kubernetes.Interface, namespace, name string) (*nadv1.NetworkAttachmentDefinition, error) {
	path := fmt.Sprintf("/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s", namespace, name)
	rawNetworkAttachmentDefinition, err := client.ExtensionsV1beta1().RESTClient().Get().AbsPath(path).DoRaw(context.TODO())
	if err != nil {
		err := errors.Wrapf(err, "could not get Network Attachment Definition %s/%s", namespace, name)
		glog.Error(err)
		return nil, err
	}

	networkAttachmentDefinition := nadv1.NetworkAttachmentDefinition{}
	err = json.Unmarshal(rawNetworkAttachmentDefinition, &networkAttachmentDefinition)

	return &networkAttachmentDefinition, err
}

func parseNetworkAttachDefinition(client kubernetes.Interface, net *multus.NetworkSelectionElement, reqs map[string]int64,
	nsMap map[string]string) (map[string]int64, map[string]string, error) {
	/* for each network in annotation ask API server for network-attachment-definition */
	nadCache := netcache.Get()
	if nadCache == nil {
		return reqs, nsMap, fmt.Errorf("failed to get NAD cache. Create it first")
	}
	annotationsMap := nadCache.Get(net.Namespace, net.Name)
	if annotationsMap == nil {
		glog.Infof("cache entry not found, retrieving network attachment definition '%s/%s' from api server", net.Namespace, net.Name)
		networkAttachmentDefinition, err := getNetworkAttachmentDefinition(client, net.Namespace, net.Name)
		if err != nil {
			/* if doesn't exist: deny pod */
			reason := errors.Wrapf(err, "could not find network attachment definition '%s/%s'", net.Namespace, net.Name)
			glog.Error(reason)
			return reqs, nsMap, reason
		}
		annotationsMap = networkAttachmentDefinition.GetAnnotations()
	}
	glog.Infof("network attachment definition '%s/%s' found", net.Namespace, net.Name)

	/* network object exists, so check if it contains resourceName annotation */
	for _, networkResourceNameKey := range resourceNameKeys {
		if resourceName, exists := annotationsMap[networkResourceNameKey]; exists {
			/* add resource to map/increment if it was already there */
			reqs[resourceName]++
			glog.Infof("resource '%s' needs to be requested for network '%s/%s'", resourceName, net.Namespace, net.Name)
		} else {
			glog.Infof("network '%s/%s' doesn't use custom resources, skipping...", net.Namespace, net.Name)
		}
	}

	/* parse the net-attach-def annotations for node selector label and add it to the desiredNsMap */
	if ns, exists := annotationsMap[nodeSelectorKey]; exists {
		nsNameValue := strings.Split(ns, "=")
		nsNameValueLen := len(nsNameValue)
		if nsNameValueLen > 2 {
			reason := fmt.Errorf("node selector in net-attach-def %s has more than one label", net.Name)
			glog.Error(reason)
			return reqs, nsMap, reason
		} else if nsNameValueLen == 2 {
			nsMap[strings.TrimSpace(nsNameValue[0])] = strings.TrimSpace(nsNameValue[1])
		} else {
			nsMap[strings.TrimSpace(ns)] = ""
		}
	}

	return reqs, nsMap, nil
}

func handleValidationError(w http.ResponseWriter, ar *v1beta1.AdmissionReview, orgErr error) {
	err := prepareAdmissionReviewResponse(false, orgErr.Error(), ar)
	if err != nil {
		err := errors.Wrap(err, "error preparing AdmissionResponse")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeResponse(w, ar)
}

func writeResponse(w http.ResponseWriter, ar *v1beta1.AdmissionReview) {
	glog.Infof("sending response to the Kubernetes API server")
	resp, _ := json.Marshal(ar)
	_, err := w.Write(resp)
	if err != nil {
		glog.Warningf("write response failed: %s", err.Error())
	}
}

func deserializeNetworkAttachmentDefinition(ar *v1beta1.AdmissionReview) (nadv1.NetworkAttachmentDefinition, error) {
	/* unmarshal NetworkAttachmentDefinition from AdmissionReview request */
	netAttachDef := nadv1.NetworkAttachmentDefinition{}
	err := json.Unmarshal(ar.Request.Object.Raw, &netAttachDef)
	return netAttachDef, err
}
