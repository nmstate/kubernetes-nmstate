package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func badDefaultGw(address string, nic string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      dhcp: false
      enabled: true
      address:
        - ip: %s
          prefix-length: 24
routes:
  config:
    - destination: 0.0.0.0/0
      metric: 150
      next-hop-address: 192.0.2.1
      next-hop-interface: %s
`, nic, address, nic))
}

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
				By("Check that desiredState is applied")
				interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				By(fmt.Sprintf("Check that %s is rolled back again", bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})
	})
	Context("when connectivity to default gw is lost after state configuration", func() {
		BeforeEach(func() {
			By("Configure a invalid default gw")
			for _, node := range nodes {
				var address string
				Eventually(func() string {
					address = ipv4Address(node, "eth0")
					return address
				}, ReadTimeout, ReadInterval).ShouldNot(BeEmpty())
				updateDesiredStateAtNode(node, badDefaultGw(address, "eth0"))
			}
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to a good gw configuration", func() {
			for _, node := range nodes {
				By("Check that desiredState is applied")
				Eventually(func() bool {
					return dhcpFlag(node, "eth0")
				}, ReadTimeout, ReadInterval).Should(BeFalse())

				By("Check that eth0 is rolled back")
				Eventually(func() bool {
					return dhcpFlag(node, "eth0")
				}, ReadTimeout, ReadInterval).Should(BeTrue())
			}
		})
	})
})
