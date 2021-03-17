// +build all virtual

package e2e

import (
	"github.com/k8snetworkplumbingwg/network-resources-injector/test/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
)

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
