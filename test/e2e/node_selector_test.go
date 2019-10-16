package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeSelector", func() {
	nonexistentNodeSelector := map[string]string{"nonexistentKey": "nonexistentValue"}

	Context("when policy is set with node selector not matching any nodes", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicyAndNodeSelector(bridge1, linuxBrUp(bridge1), nonexistentNodeSelector)
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(bridge1, linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}

			deletePolicy(bridge1)
			resetDesiredStateForNodes()
		})

		It("should not update any nodes", func() {
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})

		Context("and we remove the node selector", func() {
			BeforeEach(func() {
				setDesiredStateWithPolicyAndNodeSelector(bridge1, linuxBrUp(bridge1), map[string]string{})
			})

			It("should update all nodes", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				}
			})
		})
	})
})
