package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
)

func invalidConfig(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: invalid_state
`, bridgeName))
}

var _ = Describe("NNCP Conditions", func() {
	const policyName = "test-policy"

	Context("when applying valid config", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicy(policyName, linuxBrUp(bridge1))
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(policyName, linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			deletePolicy(policyName)
		})

		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				By(fmt.Sprintf("XXX2 %v\n", nodeNetworkConfigurationPolicy(types.NamespacedName{Name: policyName})))
				policyAvailableConditionStatusEventually(policyName, node).Should(Equal(corev1.ConditionTrue))
				policyFailingConditionStatusEventually(policyName, node).Should(Equal(corev1.ConditionFalse))
			}
		})
	})

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicy(policyName, invalidConfig(bridge1))
		})

		It("should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				policyFailingConditionStatusEventually(policyName, node).Should(Equal(corev1.ConditionTrue))
				policyAvailableConditionStatusEventually(policyName, node).Should(Equal(corev1.ConditionFalse))
			}
		})
	})
})
