package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("rollback", func() {
	var (
		br1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
`)

		br1 = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port: []
`)
	)
	Context("when an error happens during state configuration", func() {
		BeforeEach(func() {
			By("Rename vlan-filtering to vlan-filtering.bak to force failure during state configuration")
			runAtPods("sudo", "mv", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
		})
		AfterEach(func() {
			By("Rename vlan-filtering.bak to vlan-filtering to leave it as it was")
			runAtPods("sudo", "mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")
			updateDesiredState(br1Absent)
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
			}
		})
		It("should rollback failed state configuration", func() {
			updateDesiredState(br1)
			for _, node := range nodes {
				By("Check that br1 has being rolled back")
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
				By("Check reconcile re-apply desiredState")
				interfacesNameForNodeEventually(node).Should(ContainElement("br1"))
				By("Check that br1 is rolled back again")
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
			}
		})
	})
})
