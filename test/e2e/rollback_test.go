package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("rollback", func() {
	Context("when an error happens during state configuration", func() {
		BeforeEach(func() {
			By("Rename vlan-filtering to vlan-filtering.bak to force failure during state configuration")
			runAtPods("sudo", "mv", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
		})
		AfterEach(func() {
			By("Rename vlan-filtering.bak to vlan-filtering to leave it as it was")
			runAtPods("sudo", "mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")
			updateDesiredState(brAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})
		It("should rollback failed state configuration", func() {
			updateDesiredState(brUpNoPorts(bridge1))
			for _, node := range nodes {
				By(fmt.Sprintf("Check that %s has being rolled back", bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				By("Check reconcile re-apply desiredState")
				interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				By(fmt.Sprintf("Check that %s is rolled back again", bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})
	})
})
