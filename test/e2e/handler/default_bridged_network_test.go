package handler

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func createBridgeOnTheDefaultInterface() nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: brext
    type: linux-bridge
    state: up
    ipv4:
      dhcp: true
      enabled: true
    bridge:
      options:
        stp:
          enabled: false
      port:
      - name: %s
`, primaryNic))
}

func resetDefaultInterface() nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: true
      dhcp: true
  - name: brext
    type: linux-bridge
    state: absent
`, primaryNic))
}

// FIXME: We have to fix this test https://github.com/nmstate/kubernetes-nmstate/issues/192
var _ = Describe("NodeNetworkConfigurationPolicy default bridged network", func() {
	var (
		DefaultNetwork = "default-network"
	)
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
				addressByNode[node] = address
			}
		})

		Context("and linux bridge is configured on top of the default interface", func() {
			BeforeEach(func() {
				By("Creating the policy")
				setDesiredStateWithPolicy(DefaultNetwork, createBridgeOnTheDefaultInterface())

				By("Waiting until the node becomes ready again")
				waitForNodesReady()

				By("Waiting for policy to be ready")
				waitForAvailablePolicy(DefaultNetwork)
			})

			AfterEach(func() {
				By(fmt.Sprintf("Removing bridge and configuring %s with dhcp", primaryNic))
				setDesiredStateWithPolicy(DefaultNetwork, resetDefaultInterface())

				By("Waiting until the node becomes ready again")
				waitForNodesReady()

				By("Wait for policy to be ready")
				waitForAvailablePolicy(DefaultNetwork)

				By(fmt.Sprintf("Check %s has the default ip address", primaryNic))
				for _, node := range nodes {
					Eventually(func() string {
						return ipv4Address(node, primaryNic)
					}, 30*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface %s address is not the original one", primaryNic))
				}

				By(fmt.Sprintf("Check %s is back as the default route interface", primaryNic))
				for _, node := range nodes {
					defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
				}

				By("Remove the policy")
				deletePolicy(DefaultNetwork)

				By("Reset desired state at all nodes")
				resetDesiredStateForNodes()
			})

			checkThatBridgeTookOverTheDefaultIP := func(nodesToCheck []string) {
				By("Verifying that the bridge obtained node's default IP")
				for _, node := range nodesToCheck {
					Eventually(func() string {
						return ipv4Address(node, "brext")
					}, 15*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface brext has not take over the %s address", primaryNic))
				}

				By("Verify that next-hop-interface for default route is brext")
				for _, node := range nodesToCheck {
					defaultRouteNextHopInterface(node).Should(Equal("brext"))

					By("Verify that VLAN configuration is done properly")
					hasVlans(node, primaryNic, 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, "brext", 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
				}
			}

			It("should successfully move default IP address on top of the bridge", func() {
				checkThatBridgeTookOverTheDefaultIP(nodes)
			})

			It("should keep the default IP address after node reboot", func() {
				nodeToReboot := nodes[0]

				err := restartNode(nodeToReboot)
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Wait for nns to be refreshed at %s", nodeToReboot))
				waitForNodeNetworkStateUpdate(nodeToReboot)

				By(fmt.Sprintf("Node %s was rebooted, verifying that bridge took over the default IP", nodeToReboot))
				checkThatBridgeTookOverTheDefaultIP([]string{nodeToReboot})
			})
		})
	})
})

func nodeReadyConditionStatus(nodeName string) (corev1.ConditionStatus, error) {
	key := types.NamespacedName{Name: nodeName}
	node := corev1.Node{}
	// We use a special context here to ensure that Client.Get does not
	// get stuck and honor the Eventually timeout and interval values.
	// It will return a timeout error in case of .Get takes more time than
	// expected so Eventually will retry after expected interval value.
	oneSecondTimeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := framework.Global.Client.Get(oneSecondTimeoutCtx, key, &node)
	if err != nil {
		return "", err
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status, nil
		}
	}
	return corev1.ConditionUnknown, nil
}

func waitForNodesReady() {
	time.Sleep(5 * time.Second)
	for _, node := range nodes {
		EventuallyWithOffset(1, func() (corev1.ConditionStatus, error) {
			return nodeReadyConditionStatus(node)
		}, 5*time.Minute, 10*time.Second).Should(Equal(corev1.ConditionTrue))
	}
}
