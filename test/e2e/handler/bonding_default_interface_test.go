/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

// TODO: When https://bugzilla.redhat.com/show_bug.cgi?id=1906307 is resolved, add firstSecondaryNic to the bond again
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
        miimon: %s
        primary: %s
      %s:
        - %s
`, bondName, fmt.Sprintf(miimonFormat, 140), primaryNic, portFieldName, primaryNic))
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
			Byf("Check %s is the default route interface and has dynamic address", primaryNic)
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
				Byf("Fetching current IP address %s", address)
				addressByNode[node] = address
			}
			Byf("Reseting state of %s", firstSecondaryNic)
			resetNicStateForNodes(firstSecondaryNic)
			Byf("Creating %s on %s and %s", bond1, primaryNic, firstSecondaryNic)
			updateDesiredStateAndWait(boundUpWithPrimaryAndSecondary(bond1))
			By("Done configuring test")

		})
		AfterEach(func() {
			Byf("Removing bond %s and configuring %s with dhcp", bond1, primaryNic)
			updateDesiredStateAndWait(bondAbsentWithPrimaryUp(bond1))

			By("Waiting until the node becomes ready again")
			for _, node := range nodes {

				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
			}

			resetDesiredStateForNodes()

			Byf("Check %s has the default ip address", primaryNic)
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
				verifyBondIsUpWithPrimaryNicIP(node, expectedBond, addressByNode[node])
			}
			// Restart only first node that it's a control-plane if other node is restarted it will stuck in NotReady state
			nodeToReboot := nodes[0]
			Byf("Reboot node %s and verify that bond still has ip of primary nic", nodeToReboot)
			restartNodeWithoutWaiting(nodeToReboot)

			By("Wait for policy re-reconciled after node reboot")
			waitForPolicyTransitionUpdate(TestPolicy)
			waitForAvailablePolicy(TestPolicy)

			Byf("Node %s was rebooted, verifying %s exists and ip was not changed", nodeToReboot, bond1)
			verifyBondIsUpWithPrimaryNicIP(nodeToReboot, expectedBond, addressByNode[nodeToReboot])
		})
	})
})

func verifyBondIsUpWithPrimaryNicIP(node string, expectedBond map[string]interface{}, ip string) {
	interfacesForNode(node).Should(ContainElement(matchingBond(expectedBond)))

	Eventually(func() string {
		return ipv4Address(node, bond1)
	}, 30*time.Second, 1*time.Second).Should(Equal(ip), fmt.Sprintf("Interface bond1 has not take over the %s address", primaryNic))
}

func resetNicStateForNodes(nicName string) {
	updateDesiredStateAndWait(ethernetNicsUp(nicName))
	deletePolicy(TestPolicy)
}
