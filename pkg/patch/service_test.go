package patch

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sync"
)

var _ = Describe("user defined injections service", func() {

	DescribeTable("Setting user-defined injections",

		func(in *corev1.ConfigMap, existing map[string]JSONOperation, out map[string]JSONOperation) {
			u := udiUpdate{udi: &UserDefinedInjections{sync.Mutex{}, map[string]JSONOperation{}}}
			u.setUserDefinedInjections(in)
			Expect(u.udi.Patches).Should(Equal(out))
		},
		Entry(
			"patch - empty config map",
			&corev1.ConfigMap{
				Data: map[string]string{},
			},
			map[string]JSONOperation{},
			map[string]JSONOperation{},
		),
		Entry(
			"patch - addtional networks annotation",
			&corev1.ConfigMap{
				Data: map[string]string{
					"nri-inject-annotation": "{\"op\": \"add\", \"path\": \"/metadata/annotations\", \"value\": {\"k8s.v1.cni.cncf.io/networks\": \"sriov-net\"}}"},
			},
			map[string]JSONOperation{},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"k8s.v1.cni.cncf.io/networks": "sriov-net"},
				},
			},
		),
		Entry(
			"patch - default network annotation",
			&corev1.ConfigMap{
				Data: map[string]string{
					"nri-inject-annotation": "{\"op\": \"add\", \"path\": \"/metadata/annotations\", \"value\": {\"v1.multus-cni.io/default-network\": \"sriov-net\"}}"},
			},
			map[string]JSONOperation{},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"v1.multus-cni.io/default-network": "sriov-net"},
				},
			},
		),
		Entry(
			"patch - non-annotation",
			&corev1.ConfigMap{
				Data: map[string]string{
					"nri-inject-labels": "{\"op\": \"add\", \"path\": \"/metadata/labels\", \"value\": {\"v1.multus-cni.io/default-network\": \"sriov-net\"}}",
				},
			},
			map[string]JSONOperation{},
			map[string]JSONOperation{},
		),
		Entry(
			"patch - remove stale entry",
			&corev1.ConfigMap{
				Data: map[string]string{},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"v1.multus-cni.io/default-network": "sriov-net"},
				},
			},
			map[string]JSONOperation{},
		),
		Entry(
			"patch - overwrite existing userDefinedInjects",
			&corev1.ConfigMap{
				Data: map[string]string{
					"nri-inject-annotation": "{\"op\": \"add\", \"path\": \"/metadata/annotations\", \"value\": {\"v1.multus-cni.io/default-network\": \"sriov-net-new\"}}"},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"v1.multus-cni.io/default-network": "sriov-net-old"},
				},
			},
			map[string]JSONOperation{
				"nri-inject-annotation": {
					Operation: "add",
					Path:      "/metadata/annotations",
					Value:     map[string]interface{}{"v1.multus-cni.io/default-network": "sriov-net-new"},
				},
			},
		),
	)
})
