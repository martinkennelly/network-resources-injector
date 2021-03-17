package e2e

import (
	"github.com/k8snetworkplumbingwg/network-resources-injector/test/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Network injection testing", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error

	Context("one network request", func() {
		BeforeEach(func() {
			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())
			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())
		})

		AfterEach(func() {
			util.DeletePod(cs.CoreV1Interface, pod, timeout)
			util.DeleteNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, testNetworkName, nad, timeout)
		})

		It("should have one limit injected", func() {
			limNo, ok := pod.Spec.Containers[0].Resources.Limits[testNetworkResName]
			Expect(ok).Should(BeTrue())
			Expect(limNo.String()).Should(Equal("1"))
		})

		It("should have one request injected", func() {
			limNo, ok := pod.Spec.Containers[0].Resources.Requests[testNetworkResName]
			Expect(ok).Should(BeTrue())
			Expect(limNo.String()).Should(Equal("1"))
		})
	})

	Context("two network requests", func() {
		BeforeEach(func() {
			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetMultiNetworks([]string{testNetworkName, testNetworkName}, *testNs, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
		})

		AfterEach(func() {
			util.DeletePod(cs.CoreV1Interface, pod, timeout)
			util.DeleteNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, testNetworkName, nad, timeout)
		})

		It("should have two limits injected", func() {
			limNo, ok := pod.Spec.Containers[0].Resources.Limits[testNetworkResName]
			Expect(ok).Should(BeTrue())
			Expect(limNo.String()).Should(Equal("2"))
		})

		It("should have two requests injected", func() {
			limNo, ok := pod.Spec.Containers[0].Resources.Requests[testNetworkResName]
			Expect(ok).Should(BeTrue())
			Expect(limNo.String()).Should(Equal("2"))
		})
	})
})

var _ = Describe("Node selector test", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error

	Context("Cluster node available", func() {

		AfterEach(func() {
			util.DeleteNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, testNetworkName, nad, timeout)
			util.DeletePod(cs.CoreV1Interface, pod, timeout)
		})

		It("POD assigned to correct cluster node, only node specified without resource name", func() {
			nad = util.GetNodeSelectorOnly(testNetworkName, *testNs, "kubernetes.io/hostname=kind-worker2")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)

			Expect(err).Should(BeNil())

			podName := defaultPodName + "-1"
			pod = util.GetOneNetwork(testNetworkName, *testNs, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())

			Expect(pod.Name).Should(Equal("nri-e2e-test-1"))
			Expect(pod.Spec.NodeName).Should(Equal("kind-worker2"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker2"}))
		})

		It("POD assigned to correct cluster node, node specified with resource name", func() {
			nad = util.GetResourceAndNodeSelector(testNetworkName, *testNs, "kubernetes.io/hostname=kind-worker2")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			podName := defaultPodName + "-2"
			pod = util.GetOneNetwork(testNetworkName, *testNs, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())

			Expect(pod.Name).Should(Equal("nri-e2e-test-2"))
			Expect(pod.Spec.NodeName).Should(Equal("kind-worker2"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker2"}))
		})

		It("POD in pending state, cluster node is not available, without resource name", func() {
			nad = util.GetNodeSelectorOnly(testNetworkName, *testNs, "kubernetes.io/hostname=kind-worker15")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			podName := defaultPodName + "-3"
			pod = util.GetOneNetwork(testNetworkName, *testNs, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)

			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(HavePrefix("timed out waiting for the condition"))

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())
			Expect(pod.Status.Phase).Should(Equal(corev1.PodPending))
			Expect(pod.Name).Should(Equal("nri-e2e-test-3"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker15"}))
		})

		It("POD in pending state, cluster node is not available, with resource name", func() {
			nad = util.GetResourceAndNodeSelector(testNetworkName, *testNs, "kubernetes.io/hostname=kind-worker10")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			podName := defaultPodName + "-4"
			pod = util.GetOneNetwork(testNetworkName, *testNs, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)

			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(HavePrefix("timed out waiting for the condition"))

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())
			Expect(pod.Status.Phase).Should(Equal(corev1.PodPending))
			Expect(pod.Name).Should(Equal("nri-e2e-test-4"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker10"}))
		})
	})
})
