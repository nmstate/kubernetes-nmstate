package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func invalidConfig(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: invalid_state
`, bridgeName))
}

var _ = Describe("NodeNetworkStateCondition", func() {
	Context("when applying valid config", func() {
		BeforeEach(func() {
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			updateDesiredState(linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable).Should(
					Equal(corev1.ConditionTrue),
				)
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing).Should(
					Equal(corev1.ConditionFalse),
				)
			}
		})
	})

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			updateDesiredState(invalidConfig(bridge1))

		})

		AfterEach(func() {
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing).Should(
					Equal(corev1.ConditionTrue),
				)
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable).Should(
					Equal(corev1.ConditionFalse),
				)
			}
		})
	})
})
