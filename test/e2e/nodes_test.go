package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var _ = Describe("Nodes", func() {
	Context("when are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				var currentStateYaml nmstatev1.State
				currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

				interfaces := interfaces(currentStateYaml)
				Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

				obtainedInterfaces := interfacesName(interfaces)
				Expect(obtainedInterfaces).To(ContainElement("eth0"))
			}
		})
		Context("and node network state is deleted", func() {
			BeforeEach(func() {
				deleteNodeNeworkStates()
			})
			It("should recreate it with currentState", func() {
				for _, node := range nodes {
					var currentStateYaml nmstatev1.State
					currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

					interfaces := interfaces(currentStateYaml)
					Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)
				}
			})
		})
		Context("and new interface is configured", func() {
			var (
				expectedDummyName = "dummy0"
			)
			BeforeEach(func() {
				createDummy(nodes, expectedDummyName)
			})
			AfterEach(func() {
				deleteDummy(nodes, expectedDummyName)
			})
			It("should update node network state with it", func() {
				for _, node := range nodes {
					Eventually(func() []string {
						var currentStateYaml nmstatev1.State
						currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

						interfaces := interfaces(currentStateYaml)
						Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

						return interfacesName(interfaces)
					}, ReadTimeout, ReadInterval).Should(ContainElement(expectedDummyName))
				}
			})
		})
	})
})
