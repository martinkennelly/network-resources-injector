// +build all virtual

package e2e

import (
	"github.com/k8snetworkplumbingwg/network-resources-injector/test/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Verify that resource and POD which consumes resource cannot be in different namespaces", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error

	Context("network attachment definition configuration error", func() {
		It("Missing network attachment definition, try to setup POD in default namespace", func() {
			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring("could not get Network Attachment Definition default/foo-network"))
		})

		It("Correct network name in CRD, but the namespace if different than in POD specification", func() {
			testNamespace := "mysterious"
			err = util.CreateNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())

			nad = util.GetResourceSelectorOnly(testNetworkName, testNamespace, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring("could not get Network Attachment Definition default/foo-network"))

			err = util.DeleteNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())
		})

		It("CRD in default namespace, and POD in custom namespace", func() {
			testNamespace := "mysterious"
			err = util.CreateNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())

			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, testNamespace, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring("could not get Network Attachment Definition mysterious/foo-network"))

			err = util.DeleteNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())
		})
	})
})

var _ = Describe("Network injection testing", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error

	Context("one network request in default namespace", func() {
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

	Context("two network requests in default namespace", func() {
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

	Context("one network request in custom namespace", func() {
		BeforeEach(func() {
			testNamespace := "mysterious"
			err = util.CreateNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())

			nad = util.GetResourceSelectorOnly(testNetworkName, testNamespace, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, testNamespace, defaultPodName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())
			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())
		})

		AfterEach(func() {
			testNamespace := "mysterious"
			err = util.DeleteNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())
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
})

var _ = Describe("Node selector test", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error

	Context("Cluster node available, default namespace", func() {
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
			Expect(pod.ObjectMeta.Namespace).Should(Equal(*testNs))
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
			Expect(pod.ObjectMeta.Namespace).Should(Equal(*testNs))
		})
	})

	Context("Cluster node not available, default namespace", func() {
		AfterEach(func() {
			util.DeleteNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, testNetworkName, nad, timeout)
			util.DeletePod(cs.CoreV1Interface, pod, timeout)
		})

		It("POD in pending state, only node selector passed without resource name", func() {
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

		It("POD in pending state, node selector and resource name in CRD", func() {
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

	Context("Cluster node available with custom namespace", func() {
		var testNamespace string

		BeforeEach(func() {
			testNamespace = "mysterious"
			err = util.CreateNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())
		})

		AfterEach(func() {
			err = util.DeleteNamespace(cs.CoreV1Interface, testNamespace, timeout)
			Expect(err).Should(BeNil())
		})

		It("POD assigned to correct cluster node, only node specified without resource name", func() {
			nad = util.GetNodeSelectorOnly(testNetworkName, testNamespace, "kubernetes.io/hostname=kind-worker2")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			podName := defaultPodName + "-5"
			pod = util.GetOneNetwork(testNetworkName, testNamespace, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())

			Expect(pod.Name).Should(Equal("nri-e2e-test-5"))
			Expect(pod.Spec.NodeName).Should(Equal("kind-worker2"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker2"}))
			Expect(pod.ObjectMeta.Namespace).Should(Equal(testNamespace))
		})

		It("POD assigned to correct cluster node, node specified with resource name", func() {
			nad = util.GetResourceAndNodeSelector(testNetworkName, testNamespace, "kubernetes.io/hostname=kind-worker2")
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			podName := defaultPodName + "-6"
			pod = util.GetOneNetwork(testNetworkName, testNamespace, podName)
			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())

			pod, err = util.UpdatePodInfo(cs.CoreV1Interface, pod, timeout)
			Expect(err).Should(BeNil())

			Expect(pod.Name).Should(Equal("nri-e2e-test-6"))
			Expect(pod.Spec.NodeName).Should(Equal("kind-worker2"))
			Expect(pod.Spec.NodeSelector).Should(Equal(map[string]string{"kubernetes.io/hostname": "kind-worker2"}))
			Expect(pod.ObjectMeta.Namespace).Should(Equal(testNamespace))
		})
	})
})

var _ = Describe("Expose hugepages via Downward API, POD in default namespace", func() {
	var pod *corev1.Pod
	var nad *cniv1.NetworkAttachmentDefinition
	var err error
	var stdoutString, stderrString string

	Context("Virtual environment, request 0 pages", func() {
		BeforeEach(func() {
			stdoutString = ""
			stderrString = ""
		})

		AfterEach(func() {
			util.DeletePod(cs.CoreV1Interface, pod, timeout)
			util.DeleteNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, testNetworkName, nad, timeout)
		})

		It("POD without annotation about resourceName, hugepages limit and memory size are defined and are equal", func() {
			nad = util.GetWithoutAnnotations(testNetworkName, *testNs)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			pod = util.AddToPodDefinitionHugePages1Gi(pod, 0, 0, 0)
			pod = util.AddToPodDefinitionMemory(pod, 0, 0, 0)

			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())

			// Check new environment variable
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "printenv")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).Should(ContainSubstring("HOSTNAME=" + pod.Name))

			// NRI will not provide Downward API for huge pages
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "ls /etc/podnetinfo")

			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(Equal("command terminated with exit code 1"))
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).Should(Equal(""))
		})

		It("POD with annotation about resourceName, hugepages limit and memory size are defined and are equal", func() {
			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			pod = util.AddToPodDefinitionHugePages1Gi(pod, 0, 0, 0)
			pod = util.AddToPodDefinitionMemory(pod, 0, 0, 0)

			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())

			// Check new environment variable
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "printenv")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).Should(ContainSubstring("HOSTNAME=" + pod.Name))

			// NRI will not provide Downward API for huge pages
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "ls /etc/podnetinfo")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_1G_limit_test"))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_1G_request_test"))
		})

		It("POD with annotation about resourceName, hugepages limit and memory size are defined and are equal", func() {
			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			pod = util.AddToPodDefinitionHugePages2Mi(pod, 0, 0, 0)
			pod = util.AddToPodDefinitionMemory(pod, 0, 0, 0)

			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())

			// Check new environment variable
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "printenv")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).Should(ContainSubstring("HOSTNAME=" + pod.Name))

			// NRI will not provide Downward API for huge pages
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "ls /etc/podnetinfo")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_2G_limit_test"))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_2G_request_test"))
		})

		It("POD with annotation about resourceName, hugepages limit and cpu request are defined", func() {
			nad = util.GetResourceSelectorOnly(testNetworkName, *testNs, testNetworkResName)
			err = util.ApplyNetworkAttachmentDefinition(networkClient.K8sCniCncfIoV1Interface, nad, timeout)
			Expect(err).Should(BeNil())

			pod = util.GetOneNetwork(testNetworkName, *testNs, defaultPodName)
			pod = util.AddToPodDefinitionHugePages1Gi(pod, 0, 0, 0)
			pod = util.AddToPodDefinitionCpuLimits(pod, 1, 0)

			err = util.CreateRunningPod(cs.CoreV1Interface, pod, timeout, interval)
			Expect(err).Should(BeNil())
			Expect(pod.Name).ShouldNot(BeNil())

			// Check new environment variable
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "printenv")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).Should(ContainSubstring("HOSTNAME=" + pod.Name))

			// NRI will not provide Downward API for huge pages
			stdoutString, stderrString, err = util.ExecuteCommand(cs.CoreV1Interface, kubeConfig, pod.Name, *testNs, pod1stContainerName, "ls /etc/podnetinfo")
			Expect(err).Should(BeNil())
			Expect(stderrString).Should(Equal(""))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_1G_limit_test"))
			Expect(stdoutString).ShouldNot(ContainSubstring("hugepages_1G_request_test"))
		})
	})
})
