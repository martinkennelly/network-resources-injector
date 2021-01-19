package patch

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("annotations", func() {

	DescribeTable("Get network selections",

		func(annotateKey string, pod corev1.Pod, patchs []JSONOperation, out string, shouldExist bool) {
			nets, exist := GetNetworkSelections(annotateKey, pod, patchs)
			Expect(exist).To(Equal(shouldExist))
			Expect(nets).Should(Equal(out))
		},
		Entry(
			"get from pod original annotation",
			"k8s.v1.cni.cncf.io/networks",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
				Spec: corev1.PodSpec{},
			},
			[]JSONOperation{
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net-user-defined"},
				},
			},
			"sriov-net",
			true,
		),
		Entry(
			"get from pod user-defined annotation",
			"k8s.v1.cni.cncf.io/networks",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"v1.multus-cni.io/default-network": "sriov-net"},
				},
				Spec: corev1.PodSpec{},
			},
			[]JSONOperation{
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net-user-defined"},
				},
			},
			"sriov-net-user-defined",
			true,
		),
		Entry(
			"get from pod user-defined annotation",
			"k8s.v1.cni.cncf.io/networks",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"v1.multus-cni.io/default-network": "sriov-net"},
				},
				Spec: corev1.PodSpec{},
			},
			[]JSONOperation{
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"v1.multus-cni.io/default-network": "sriov-net-user-defined"},
				},
			},
			"",
			false,
		),
	)
})
