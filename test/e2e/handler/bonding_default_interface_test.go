package handler

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func boundUpWithPrimaryAndSecondary(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    ipv4:
      dhcp: true
      enabled: true
    link-aggregation:
      mode: active-backup
      options:
        miimon: '140'
        primary: %s
      slaves:
        - %s
        - %s
`, bondName, primaryNic, primaryNic, firstSecondaryNic))
}

func bondAbsentWithPrimaryUp(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: absent
  - name: %s
    state: up
    type: ethernet
    ipv4:
      dhcp: true
      enabled: true
`, bondName, primaryNic))
}

var _ = Describe("NodeNetworkConfigurationPolicy bonding default interface", func() {
	Context("when there is a default interface with dynamic address", func() {
		addressByNode := map[string]string{}
		BeforeEach(func() {
			By(fmt.Sprintf("Check %s is the default route interface and has dynamic address", primaryNic))
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
				Expect(dhcpFlag(node, primaryNic)).Should(BeTrue())
			}

			By("Fetching current IP address")
			for _, node := range nodes {
				address := ""
				Eventually(func() string {
					address = ipv4Address(node, primaryNic)
					return address
				}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), fmt.Sprintf("Interface %s has no ipv4 address", primaryNic))
				By(fmt.Sprintf("Fetching current IP address %s", address))
				addressByNode[node] = address
			}
			By(fmt.Sprintf("Reseting state of %s", firstSecondaryNic))
			resetNicStateForNodes(firstSecondaryNic)
			By(fmt.Sprintf("Creating %s on %s and %s", bond1, primaryNic, firstSecondaryNic))
			updateDesiredStateAndWait(boundUpWithPrimaryAndSecondary(bond1))
			By("Done configuring test")

		})
		AfterEach(func() {
			By(fmt.Sprintf("Removing bond %s and configuring %s with dhcp", bond1, primaryNic))
			updateDesiredStateAndWait(bondAbsentWithPrimaryUp(bond1))

			By("Waiting until the node becomes ready again")
			for _, node := range nodes {

				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
			}

			resetDesiredStateForNodes()

			By(fmt.Sprintf("Check %s has the default ip address", primaryNic))
			for _, node := range nodes {
				Eventually(func() string {
					return ipv4Address(node, primaryNic)
				}, 30*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface %s address is not the original one", primaryNic))
			}

		})

		It("should successfully move default IP address on top of the bond", func() {
			var (
				expectedBond = interfaceByName(interfaces(boundUpWithPrimaryAndSecondary(bond1)), bond1)
			)

			By("Checking that bond was configured and obtained the same IP address")
			for _, node := range nodes {
				verifyBondIsUpWithPrimaryNicIp(node, expectedBond, addressByNode[node])
			}
			// Restart only first node that it master if other node is restarted it will stuck in NotReady state
			nodeToReboot := nodes[0]
			By(fmt.Sprintf("Reboot node %s and verify that bond still has ip of primary nic", nodeToReboot))
			err := restartNode(nodeToReboot)
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("Wait for nns to be refreshed at %s", nodeToReboot))
			waitForNodeNetworkStateUpdate(nodeToReboot)

			By(fmt.Sprintf("Node %s was rebooted, verifying %s exists and ip was not changed", nodeToReboot, bond1))
			verifyBondIsUpWithPrimaryNicIp(nodeToReboot, expectedBond, addressByNode[nodeToReboot])
		})
	})
})

func verifyBondIsUpWithPrimaryNicIp(node string, expectedBond map[string]interface{}, ip string) {
	interfacesForNode(node).Should(ContainElement(matchingBond(expectedBond)))

	Eventually(func() string {
		return ipv4Address(node, bond1)
	}, 30*time.Second, 1*time.Second).Should(Equal(ip), fmt.Sprintf("Interface bond1 has not take over the %s address", primaryNic))
}

func resetNicStateForNodes(nicName string) {
	updateDesiredStateAndWait(ethernetNicsUp(nicName))
	deletePolicy(TestPolicy)
}
