package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	runner "github.com/nmstate/kubernetes-nmstate/test/runner"
)

var _ = Describe("NodeNetworkStateCondition", func() {
	Context("when cluster is initialized", func() {
		BeforeEach(func() {
			deleteNodeNeworkStates()
		})
		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				stateConditionsStatusEventually(node).Should(SatisfyAll(
					containStateAvailable(),
					containStateNotDegraded(),
				))
			}
		})
	})

	Context("when nmstatectl show fails", func() {
		BeforeEach(func() {
			By("Rename nmstatectl to nmstatectl.bak to force failure during nmstatectl show")
			runner.RunAtPods("mv", "/usr/bin/nmstatectl", "/usr/bin/nmstatectl.bak")
			deleteNodeNeworkStates()
		})
		AfterEach(func() {
			By("Rename nmstatectl.bak to nmstatectl to have functional nmstatectl")
			runner.RunAtPods("mv", "/usr/bin/nmstatectl.bak", "/usr/bin/nmstatectl")
		})
		It("should fail", func() {
			for _, node := range nodes {
				stateConditionsStatusEventually(node).Should(SatisfyAll(
					containStateNotAvailable(),
					containStateDegraded(),
				))
			}
		})
	})
})
