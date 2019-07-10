package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodes", func() {
	Context("when are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				interfacesNameForNode(node).Should(ContainElement("eth0"))
			}
		})
		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable, corev1.ConditionTrue)
			}
		})
		Context("and node network state is deleted", func() {
			BeforeEach(func() {
				deleteNodeNeworkStates()
			})
			It("should recreate it with currentState", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).Should(ContainElement("eth0"))
				}
			})
			FIt("should have Initialized ConditionType set to true", func() {
				for _, node := range nodes {
					checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionInitialized, corev1.ConditionTrue)
				}
			})
		})
		Context("and new interface is configured", func() {
			var (
				expectedDummyName = "dummy0"
			)
			BeforeEach(func() {
				createDummyAtNodes(expectedDummyName)
			})
			AfterEach(func() {
				deleteConnectionAtNodes(expectedDummyName)
			})
			It("should update node network state with it", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).Should(ContainElement(expectedDummyName))
				}
			})
		})
	})
})
