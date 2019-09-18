package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tidwall/gjson"

	yaml "sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkConfigurationPolicy default bridged network", func() {
	createBridgeOnTheDefaultInterface := nmstatev1alpha1.State(`interfaces:
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
      - name: eth0
`)
	resetDefaultInterface := nmstatev1alpha1.State(`interfaces:
  - name: eth0
    type: ethernet
    state: up
    ipv4:
      enabled: true
      dhcp: true
  - name: brext
    type: linux-bridge
    state: absent
`)
	// FIXME: This is a pending spec since we have to discover why the
	//        cluster never goes back at kubevirtci provider
	XContext("when there is a default interface with dynamic address", func() {
		addressByNode := map[string]string{}
		BeforeEach(func() {
			By("Check eth0 is the default route interface and has dynamic address")
			for _, node := range nodes {
				Expect(defaultRouteNextHopInterface(node)).To(Equal("eth0"))
				Expect(dhcpFlag(node, "eth0")).To(BeTrue())
			}

			By("Fetching current IP address")
			for _, node := range nodes {
				address := ""
				Eventually(func() string {
					return ipv4Address(node, "eth0")
				}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "Interface eth0 has no ipv4 address")
				addressByNode[node] = address
			}
		})
		AfterEach(func() {
			By("Removing bridge and configuring eth0 with dhcp")
			setDesiredStateWithPolicy("default-network", resetDefaultInterface)

			By("Check eth0 has the default ip address")
			for _, node := range nodes {
				Eventually(func() string {
					return ipv4Address(node, "eth0")
				}, 15*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), "Interface eth0 address is not the original one")
			}

			By("Check eth0 is back as the default route interface")
			for _, node := range nodes {
				Eventually(func() string {
					return defaultRouteNextHopInterface(node)
				}, 30*time.Second, 1*time.Second).Should(Equal("eth0"))
			}

			By("Waiting until the node becomes ready again")
			waitForNodesReady()

			By("Remove the policy")
			deletePolicy("default-network")
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
				}, 15*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), "Interface brext has not take over the eth0 address")
			}

			By("Verify that next-hop-interface for default route is brext")
			for _, node := range nodes {
				Eventually(func() string {
					return defaultRouteNextHopInterface(node)
				}, 30*time.Second, 1*time.Second).Should(Equal("brext"))

				By("Verify that VLAN configuration is done properly")
				hasVlans(node, "eth0", 2, 4094).Should(Succeed())
				hasVlans(node, "brext", 1, 1).Should(Succeed())
			}
		})
	})
})

func currentStateJSON(node string) []byte {
	key := types.NamespacedName{Name: node}
	currentState := nodeNetworkState(key).Status.CurrentState
	currentStateJson, err := yaml.YAMLToJSON([]byte(currentState))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return currentStateJson
}

func ipv4Address(node string, name string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.address.0.ip", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func defaultRouteNextHopInterface(node string) string {
	path := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-interface"
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func dhcpFlag(node string, name string) bool {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.dhcp", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).Bool()
}

func nodeReadyConditionStatus(nodeName string) (corev1.ConditionStatus, error) {
	key := types.NamespacedName{Name: nodeName}
	node := corev1.Node{}
	err := framework.Global.Client.Get(context.TODO(), key, &node)
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
	for _, node := range nodes {
		Eventually(func() (corev1.ConditionStatus, error) {
			return nodeReadyConditionStatus(node)
		}, 15*time.Second, 1*time.Second).Should(Equal(corev1.ConditionTrue))
	}
}
