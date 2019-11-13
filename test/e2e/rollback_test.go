package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

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
	BeforeEach(func() {
		By("Check Enactment is not at failing condition")
		for _, node := range nodes {
			checkEnactmentConditionConsistently(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing).
				Should(Equal(corev1.ConditionFalse), "NodeNetworkState should not be Failing before test")
		}
	})
	Context("when an error happens during state configuration", func() {
		BeforeEach(func() {
			By("Rename vlan-filtering to vlan-filtering.bak to force failure during state configuration")
			runAtPods("mv", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
		})
		AfterEach(func() {
			By("Rename vlan-filtering.bak to vlan-filtering to leave it as it was")
			runAtPods("mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")
			updateDesiredState(linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			resetDesiredStateForNodes()
		})
		It("should rollback failed state configuration", func() {
			updateDesiredState(linuxBrUpNoPorts(bridge1))
			for _, node := range nodes {
				By("Wait for reconcile to fail")
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing).
					Should(Equal(corev1.ConditionTrue), "NodeNetworkState should be failing after rollback")

				By(fmt.Sprintf("Check that %s has being rolled back", bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))

				By(fmt.Sprintf("Check that %s continue with rolled back state", bridge1))
				interfacesNameForNodeConsistently(node).ShouldNot(ContainElement(bridge1))
			}
		})
	})
	Context("when connectivity to default gw is lost after state configuration", func() {
		BeforeEach(func() {
			By("Configure a invalid default gw")
			for _, node := range nodes {
				var address string
				Eventually(func() string {
					address = ipv4Address(node, *primaryNic)
					return address
				}, ReadTimeout, ReadInterval).ShouldNot(BeEmpty())
				updateDesiredStateAtNode(node, badDefaultGw(address, *primaryNic))
			}
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to a good gw configuration", func() {
			for _, node := range nodes {
				By("Wait for reconcile to fail")
				checkEnactmentConditionEventually(node, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing).
					Should(Equal(corev1.ConditionTrue), "NodeNetworkState should be failing after rollback")

				By(fmt.Sprintf("Check that %s is rolled back", *primaryNic))
				Eventually(func() bool {
					return dhcpFlag(node, *primaryNic)
				}, ReadTimeout, ReadInterval).Should(BeTrue(), "DHCP flag hasn't rollback to true")

				By(fmt.Sprintf("Check that %s continue with rolled back state", *primaryNic))
				Consistently(func() bool {
					return dhcpFlag(node, *primaryNic)
				}, 5*time.Second, 1*time.Second).Should(BeTrue(), "DHCP flag has change to false")

			}
		})
	})
})
