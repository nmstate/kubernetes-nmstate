package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tidwall/gjson"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

// FIXME: We have to fix this test https://github.com/nmstate/kubernetes-nmstate/issues/192
var _ = Describe("NodeNetworkConfigurationPolicy default bridged network", func() {
	createBridgeOnTheDefaultInterface := nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
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
`, *primaryNic))
	resetDefaultInterface := nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: true
      dhcp: true
  - name: brext
    type: linux-bridge
    state: absent
`, *primaryNic))
	Context("when there is a default interface with dynamic address", func() {
		addressByNode := map[string]string{}
		BeforeEach(func() {
			By(fmt.Sprintf("Check %s is the default route interface and has dynamic address", *primaryNic))
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal(*primaryNic))
				Expect(dhcpFlag(node, *primaryNic)).Should(BeTrue())
			}

			By("Fetching current IP address")
			for _, node := range nodes {
				address := ""
				Eventually(func() string {
					address = ipv4Address(node, *primaryNic)
					return address
				}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), fmt.Sprintf("Interface %s has no ipv4 address", *primaryNic))
				addressByNode[node] = address
			}
		})
		AfterEach(func() {
			By(fmt.Sprintf("Removing bridge and configuring %s with dhcp", *primaryNic))
			setDesiredStateWithPolicy("default-network", resetDefaultInterface)

			By("Waiting until the node becomes ready again")
			waitForNodesReady()

			By(fmt.Sprintf("Check %s has the default ip address", *primaryNic))
			for _, node := range nodes {
				Eventually(func() string {
					return ipv4Address(node, *primaryNic)
				}, 30*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface %s address is not the original one", *primaryNic))
			}

			By(fmt.Sprintf("Check %s is back as the default route interface", *primaryNic))
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal(*primaryNic))
			}

			By("Remove the policy")
			deletePolicy("default-network")

			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("should successfully move default IP address on top of the bridge", func() {
			By("Creating the policy")
			setDesiredStateWithPolicy("default-network", createBridgeOnTheDefaultInterface)

			By("Waiting until the node becomes ready again")
			waitForNodesReady()

			By("Checking that obtained the same IP address")
			for _, node := range nodes {
				Eventually(func() string {
					return ipv4Address(node, "brext")
				}, 15*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), fmt.Sprintf("Interface brext has not take over the %s address", *primaryNic))
			}

			By("Verify that next-hop-interface for default route is brext")
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal("brext"))

				By("Verify that VLAN configuration is done properly")
				hasVlans(node, *primaryNic, 2, 4094).Should(Succeed())
				vlansCardinality(node, "brext").Should(Equal(0))
			}
		})
	})
})

func defaultRouteNextHopInterface(node string) AsyncAssertion {
	return Eventually(func() string {
		path := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-interface"
		return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
	}, 15*time.Second, 1*time.Second)
}

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
