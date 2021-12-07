package handler

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("NodeNetworkConfigurationPolicy default bridged network with nmpolicy", func() {
	var (
		DefaultNetwork = "default-network"
	)
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
				addressByNode[node] = address
			}
		})

		Context("and linux bridge is configured on top of the default interface", func() {
			BeforeEach(func() {
				By("Creating the policy")
				bridgeOnTheDefaultInterfaceState := nmstate.NewState(`interfaces:
  - name: brext
    type: linux-bridge
    state: up
    mac-address: "{{ capture.base-iface.interfaces.0.mac-address }}"
    ipv4:
      dhcp: true
      enabled: true
    bridge:
      options:
        stp:
          enabled: false
      port:
      - name: "{{ capture.base-iface.interfaces.0.name }}"
`)
				capture := map[string]string{
					"default-gw": `routes.running.destination=="0.0.0.0/0"`,
					"base-iface": `interfaces.name==capture.default-gw.routes.running.0.next-hop-interface`,
				}
				setDesiredStateWithPolicyAndCapture(DefaultNetwork, bridgeOnTheDefaultInterfaceState, capture)

				By("Waiting until the node becomes ready again")
				waitForNodesReady()

				By("Waiting for policy to be ready")
				waitForAvailablePolicy(DefaultNetwork)
			})

			AfterEach(func() {
				resetDefaultInterfaceState := nmstate.NewState(`interfaces:
  - name: "{{ capture.brext-bridge.interfaces.0.bridge.port.0.name }}"
    type: ethernet
    state: up
    ipv4:
      enabled: true
      dhcp: true
  - name: brext
    type: linux-bridge
    state: absent
`)

				capture := map[string]string{
					"brext-bridge": `interfaces.name=="brext"`,
				}

				Byf("Removing bridge and configuring %s with dhcp", primaryNic)
				setDesiredStateWithPolicyAndCapture(DefaultNetwork, resetDefaultInterfaceState, capture)

				By("Waiting until the node becomes ready again")
				waitForNodesReady()

				By("Wait for policy to be ready")
				waitForAvailablePolicy(DefaultNetwork)

				Byf("Check %s has the default ip address", primaryNic)
				for _, node := range nodes {
					Eventually(func() string {
						return ipv4Address(node, primaryNic)
					}, 30*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface %s address is not the original one", primaryNic))
				}

				Byf("Check %s is back as the default route interface", primaryNic)
				for _, node := range nodes {
					defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
				}

				By("Remove the policy")
				deletePolicy(DefaultNetwork)

				By("Reset desired state at all nodes")
				resetDesiredStateForNodes()
			})

			It("should successfully move default IP address on top of the bridge", func() {
				checkThatBridgeTookOverTheDefaultIP(nodes, "brext", addressByNode)
			})

			It("should keep the default IP address after node reboot", func() {
				nodeToReboot := nodes[0]

				err := restartNode(nodeToReboot)
				Expect(err).ToNot(HaveOccurred())

				By("Wait for policy re-reconciled after node reboot")
				waitForPolicyTransitionUpdate(DefaultNetwork)
				waitForAvailablePolicy(DefaultNetwork)

				Byf("Node %s was rebooted, verifying that bridge took over the default IP", nodeToReboot)
				checkThatBridgeTookOverTheDefaultIP([]string{nodeToReboot}, "brext", addressByNode)
			})
		})
	})
})
