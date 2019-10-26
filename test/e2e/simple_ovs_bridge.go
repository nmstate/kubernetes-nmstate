package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Simple OVS bridge", func() {
	const policyName = "test-policy"

	Context("when desiredState is configured with an ovs bridge up", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicy(policyName, ovsBrUp(bridge1))
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(policyName, ovsBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			deletePolicy(policyName)
		})

		It("should have the ovs bridge at currentState", func() {
			for _, node := range nodes {
				interfacesForNode(node).Should(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", bridge1),
					HaveKeyWithValue("type", "ovs-bridge"),
					HaveKeyWithValue("state", "up"),
				)))
			}
		})
	})

	Context("when desiredState is configured with an ovs bridge with internal port up", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicy(policyName, ovsBrWithInternalInterface(bridge1))
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(policyName, ovsBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			deletePolicy(policyName)
		})

		It("should have the ovs bridge at currentState", func() {
			for _, node := range nodes {
				interfacesForNode(node).Should(SatisfyAll(
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", bridge1),
						HaveKeyWithValue("type", "ovs-bridge"),
						HaveKeyWithValue("state", "up"),
					)),
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", "ovs0"),
						HaveKeyWithValue("type", "ovs-interface"),
						HaveKeyWithValue("state", "up"),
					)),
				))
			}
		})
	})
})
