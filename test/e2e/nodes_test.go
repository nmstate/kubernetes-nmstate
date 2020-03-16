package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodes", func() {
	Context("when are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(primaryNic))
			}
		})
		Context("and node network state is deleted", func() {
			BeforeEach(func() {
				deleteNodeNeworkStates()
			})
			It("should recreate it with currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(primaryNic))
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
				By("Make sure the dummy interface gets deleted")
				for _, node := range nodes {
					for _, iface := range interfacesNameForNode(node) {
						if iface == expectedDummyName {
							deleteDeviceAtNode(node, expectedDummyName)
						}
					}
				}
			})
			// CNV-3794
			It("should update node network state with it", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(expectedDummyName))
				}
			})
		})
	})
})
