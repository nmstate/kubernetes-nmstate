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
	const policyName = "test-policy"

	Context("when an error happens during state configuration", func() {
		BeforeEach(func() {
			By("Rename vlan-filtering to vlan-filtering.bak to force failure during state configuration")
			runAtPods("mv", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
		})

		AfterEach(func() {
			By("Rename vlan-filtering.bak to vlan-filtering to leave it as it was")
			runAtPods("mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")
			setDesiredStateWithPolicy(policyName, linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			deletePolicy(policyName)
		})

		It("should rollback failed state configuration", func() {
			setDesiredStateWithPolicy(policyName, linuxBrUpNoPorts(bridge1))

			for _, node := range nodes {
				By("Wait for reconcile to fail")
				policyFailingConditionStatusEventually(policyName, node).
					Should(Equal(corev1.ConditionTrue), "Policy should be marked as failing after rollback")

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
				setDesiredStateWithPolicyAndNodeSelector(fmt.Sprintf("%s-%s", policyName, node), badDefaultGw(address, *primaryNic), map[string]string{"kubernetes.io/hostname": node})
			}
		})

		AfterEach(func() {
			for _, node := range nodes {
				deletePolicy(fmt.Sprintf("%s-%s", policyName, node))
			}
		})

		// TODO: maybe the controller gets restarted during default gw configuration?
		// it configures the bridge but logs contain nothing
		// TODO: check if there is more restarts after the conf, keep checking logs
		It("should rollback to a good gw configuration", func() {
			for _, node := range nodes {
				By("Wait for reconcile to fail")
				policyFailingConditionStatusEventually(fmt.Sprintf("%s-%s", policyName, node), node).
					Should(Equal(corev1.ConditionTrue), "Policy should be marked as failing after rollback")

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
