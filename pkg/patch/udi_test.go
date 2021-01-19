package patch

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("user defined injections", func() {
	DescribeTable("Create user-defined patches",

		func(pod corev1.Pod, userDefinedInjectPatches map[string]JSONOperation, out []JSONOperation) {
			udi := &UserDefinedInjections{}
			udi.Patches = userDefinedInjectPatches
			appliedPatches, _ := udi.CreateCustomizedPatch(pod)
			Expect(appliedPatches).Should(Equal(out))
		},
		Entry(
			"match pod label",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"nri-inject-annotation": "true"},
				},
				Spec: corev1.PodSpec{},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
			},
			[]JSONOperation{
				{
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
			},
		),
		Entry(
			"doesn't match pod label value",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"nri-inject-annotation": "false"},
				},
				Spec: corev1.PodSpec{},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
			},
			nil,
		),
		Entry(
			"doesn't match pod label key",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"nri-inject-labels": "true"},
				},
				Spec: corev1.PodSpec{},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
			},
			nil,
		),
	)
})
