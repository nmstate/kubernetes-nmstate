package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	runner "github.com/nmstate/kubernetes-nmstate/test/runner"
)

// We cannot change routes at nmstate if the interface is with dhcp true
// that's why we need to set it static with the same ip it has previously.
func badDefaultGw(address string, nic string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
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

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component]rollback", func() {
	Context("when an error happens during state configuration", func() {
		BeforeEach(func() {
			By("Rename vlan-filtering to vlan-filtering.bak to force failure during state configuration")
			runner.RunAtPods("mv", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
		})
		AfterEach(func() {
			By("Rename vlan-filtering.bak to vlan-filtering to leave it as it was")
			runner.RunAtPods("mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")
			updateDesiredState(linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})
		It("should rollback failed state configuration", func() {
			updateDesiredState(linuxBrUpNoPorts(bridge1))
			By("Wait for reconcile to fail")
			waitForDegradedTestPolicy()
			for _, node := range nodes {
				By(fmt.Sprintf("Check that %s has being rolled back", bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))

				By(fmt.Sprintf("Check that %s continue with rolled back state", bridge1))
				interfacesNameForNodeConsistently(node).ShouldNot(ContainElement(bridge1))
			}
		})
	})
	// This spec is done only at first node since policy has to be different
	// per node (ip addresses has to be different at cluster).
	Context("[test_id:3793]when connectivity to default gw is lost after state configuration", func() {
		BeforeEach(func() {
			By("Configure a invalid default gw")
			var address string
			Eventually(func() string {
				address = ipv4Address(nodes[0], primaryNic)
				return address
			}, ReadTimeout, ReadInterval).ShouldNot(BeEmpty())
			updateDesiredStateAtNode(nodes[0], badDefaultGw(address, primaryNic))
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to a good gw configuration", func() {
			By("Wait for reconcile to fail")
			waitForDegradedTestPolicy()
			By(fmt.Sprintf("Check that %s is rolled back", primaryNic))
			Eventually(func() bool {
				return dhcpFlag(nodes[0], primaryNic)
			}, ReadTimeout, ReadInterval).Should(BeTrue(), "DHCP flag hasn't rollback to true")

			By(fmt.Sprintf("Check that %s continue with rolled back state", primaryNic))
			Consistently(func() bool {
				return dhcpFlag(nodes[0], primaryNic)
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "DHCP flag has change to false")
		})
	})
})
