package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkStateCondition", func() {
	Context("when cluster is initialized", func() {
		BeforeEach(func() {
			deleteNodeNeworkStates()
		})
		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable).Should(
					Equal(corev1.ConditionTrue),
				)
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionFailing).Should(
					Equal(corev1.ConditionFalse),
				)
			}
		})
	})

	Context("when nmstatectl show fails", func() {
		BeforeEach(func() {
			By("Rename nmstatectl to nmstatectl.bak to force failure during nmstatectl show")
			runAtPods("mv", "/usr/bin/nmstatectl", "/usr/bin/nmstatectl.bak")
			deleteNodeNeworkStates()
		})
		AfterEach(func() {
			By("Rename nmstatectl.bak to nmstatectl to have functional nmstatectl")
			runAtPods("mv", "/usr/bin/nmstatectl.bak", "/usr/bin/nmstatectl")
		})
		It("should fail", func() {
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionFailing).Should(
					Equal(corev1.ConditionTrue),
				)
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable).Should(
					Equal(corev1.ConditionFalse),
				)
			}
		})
	})
})
