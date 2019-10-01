package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkStateCondition", func() {
	var (
		br1Up = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth1
`)
		br1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
`)
		invalidConfig = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: invalid_state
`)
	)
	Context("when applying valid config", func() {
		BeforeEach(func() {
			updateDesiredState(br1Up)
		})
		AfterEach(func() {
			updateDesiredState(br1Absent)
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
			}
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

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			updateDesiredState(invalidConfig)
		})
		It("should have Failing ConditionType set to true", func() {
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
