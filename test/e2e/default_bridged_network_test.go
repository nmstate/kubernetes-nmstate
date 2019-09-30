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

// FIXME: We have to fix this test https://github.com/nmstate/kubernetes-nmstate/issues/192
var _ = PDescribe("NodeNetworkConfigurationPolicy default bridged network", func() {
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
	Context("when there is a default interface with dynamic address", func() {
		addressByNode := map[string]string{}
		BeforeEach(func() {
			By("Check eth0 is the default route interface and has dynamic address")
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal("eth0"))
				Expect(dhcpFlag(node, "eth0")).Should(BeTrue())
			}

			By("Fetching current IP address")
			for _, node := range nodes {
				address := ""
				Eventually(func() string {
					address = ipv4Address(node, "eth0")
					return address
				}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "Interface eth0 has no ipv4 address")
				addressByNode[node] = address
			}
		})
		AfterEach(func() {
			By("Removing bridge and configuring eth0 with dhcp")
			setDesiredStateWithPolicy("default-network", resetDefaultInterface)

			By("Waiting until the node becomes ready again")
			waitForNodesReady()

			By("Check eth0 has the default ip address")
			for _, node := range nodes {
				Eventually(func() string {
					return ipv4Address(node, "eth0")
				}, 15*time.Second, 1*time.Second).Should(Equal(addressByNode[node]), "Interface eth0 address is not the original one")
			}

			By("Check eth0 is back as the default route interface")
			for _, node := range nodes {
				defaultRouteNextHopInterface(node).Should(Equal("eth0"))
			}

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
				defaultRouteNextHopInterface(node).Should(Equal("brext"))

				By("Verify that VLAN configuration is done properly")
				hasVlans(node, "eth0", 2, 4094).Should(Succeed())
				vlansCardinality(node, "brext").Should(Equal(0))
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

func defaultRouteNextHopInterface(node string) AsyncAssertion {
	return Eventually(func() string {
		path := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-interface"
		return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
	}, 15*time.Second, 1*time.Second)
}

func dhcpFlag(node string, name string) bool {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.dhcp", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).Bool()
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
