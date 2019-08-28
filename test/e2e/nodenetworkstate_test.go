package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/types"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func hasVlans(result gjson.Result, maxVlan int) {
	vlans := result.Array()
	ExpectWithOffset(1, vlans).To(HaveLen(maxVlan))
	for i, vlan := range vlans {
		Expect(vlan.Get("vlan").Int()).To(Equal(int64(i + 1)))
	}
}

var _ = Describe("NodeNetworkState", func() {
	var (
		bond1Up = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: bond1
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - eth1
      options:
        miimon: '120'
`)
		br1Up = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)
		brextWorkaroundUp = nmstatev1alpha1.State(`interfaces:
- bridge:
    options:
      vlan-filtering: true
      vlans:
      - vlan-range-max: 4094
        vlan-range-min: 1
    port:
    - name: eth0
      vlans:
      - vlan-range-max: 4094
        vlan-range-min: 1
  ipv4:
    dhcp: true
    enabled: true
  ipv6:
    dhcp: true
    enabled: true
  name: brext
  state: up
  type: linux-bridge
`)
		br1WithBond1Up = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: bond1
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - eth1
      options:
        miimon: '120'
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: bond1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)

		br1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
  - name: eth1
    type: ethernet
    state: absent
`)
	)
	Context("when desiredState is configured", func() {
		Context("with a linux bridge up", func() {
			BeforeEach(func() {
				updateDesiredState(br1Up)
			})
			AfterEach(func() {

				// First we clean desired state if we
				// don't do that nmstate recreates the bridge
				resetDesiredStateForNodes()

				// TODO: Add status conditions to ensure that
				//       it has being really reset so we can
				//       remove this ugly sleep
				time.Sleep(1 * time.Second)

				// Let's clean the bridge directly in the node
				// bypassing nmstate
				deleteConnectionAtNodes("eth1")
				deleteConnectionAtNodes("br1")
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).Should(ContainElement("br1"))
				}
			})
		})
		FContext("with a linux bridge workaround (dhcp+trunk) up", func() {
			// TODO: setup on the default interface
			BeforeEach(func() {
				updateDesiredState(brextWorkaroundUp)
			})
			AfterEach(func() {
				// First we clean desired state if we
				// don't do that nmstate recreates the bridge
				resetDesiredStateForNodes()

				// TODO: Add status conditions to ensure that
				//       it has being really reset so we can
				//       remove this ugly sleep
				time.Sleep(1 * time.Second)

				// Let's clean the bridge directly in the node
				// bypassing nmstate
				deleteConnectionAtNodes("eth0")
				deleteConnectionAtNodes("brext")
				// TODO: wait till eventually gets back connectivity
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					// TODO: wait until gets connectivity to the node
					Eventually(func() error {
						return framework.Global.Client.Get(context.TODO(), types.NamespacedName{Name: node}, &nmstatev1alpha1.NodeNetworkState{})
					}, 5*time.Minute, 10*time.Second).ShouldNot(HaveOccurred())

					By(fmt.Sprintf("XXX: %v", node))
					Eventually(func() AsyncAssertion {
						By(fmt.Sprintf("XXX: %v", node))
						return interfacesNameForNode(node)
					}, 1*time.Minute, 5*time.Second).Should(ContainElement("brext"))
				}
			})
		})
		Context("with a linux bridge absent", func() {
			BeforeEach(func() {
				createBridgeAtNodes("br1")
				updateDesiredState(br1Absent)
			})
			AfterEach(func() {
				// If not br1 is going to be removed if created manually
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).ShouldNot(ContainElement("br1"))
				}
			})
		})
		Context("with a active-backup miimon 100 bond interface up", func() {
			BeforeEach(func() {
				updateDesiredState(bond1Up)
			})
			AfterEach(func() {

				resetDesiredStateForNodes()

				// TODO: Add status conditions to ensure that
				//       it has being really reset so we can
				//       remove this ugly sleep
				time.Sleep(1 * time.Second)

				deleteConnectionAtNodes("bond1")
				deleteConnectionAtNodes("eth1")
			})
			It("should have the bond interface at currentState", func() {
				var (
					expectedBond = interfaceByName(interfaces(bond1Up), "bond1")
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(SatisfyAll(
						HaveKeyWithValue("name", expectedBond["name"]),
						HaveKeyWithValue("type", expectedBond["type"]),
						HaveKeyWithValue("state", expectedBond["state"]),
						HaveKeyWithValue("link-aggregation", expectedBond["link-aggregation"]),
					)))
				}
			})
		})
		Context("with the bond interface as linux bridge port", func() {
			BeforeEach(func() {
				createBridgeAtNodes("br2", "eth2")
				updateDesiredState(br1WithBond1Up)
			})
			AfterEach(func() {

				resetDesiredStateForNodes()

				// TODO: Add status conditions to ensure that
				//       it has being really reset so we can
				//       remove this ugly sleep
				time.Sleep(1 * time.Second)

				deleteConnectionAtNodes("eth1")
				deleteConnectionAtNodes("eth2")
				deleteConnectionAtNodes("br1")
				deleteConnectionAtNodes("br2")
				deleteConnectionAtNodes("bond1")
			})
			It("should have the bond in the linux bridge as port at currentState", func() {
				var (
					expectedInterfaces  = interfaces(br1WithBond1Up)
					expectedBond        = interfaceByName(expectedInterfaces, "bond1")
					expectedBridge      = interfaceByName(expectedInterfaces, "br1")
					expectedBridgePorts = expectedBridge["bridge"].(map[string]interface{})["port"]
				)
				for _, node := range nodes {
					interfacesForNode(node).Should(SatisfyAll(
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedBond["name"]),
							HaveKeyWithValue("type", expectedBond["type"]),
							HaveKeyWithValue("state", expectedBond["state"]),
							HaveKeyWithValue("link-aggregation", expectedBond["link-aggregation"]),
						)),
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedBridge["name"]),
							HaveKeyWithValue("type", expectedBridge["type"]),
							HaveKeyWithValue("state", expectedBridge["state"]),
							HaveKeyWithValue("bridge", HaveKeyWithValue("port", expectedBridgePorts)),
						))))
				}
				Eventually(func() bool {
					for _, bridgeVlans := range bridgeVlansAtNodes() {
						parsedVlans := gjson.Parse(bridgeVlans)

						hasVlans(parsedVlans.Get("bond1"), 4094)
						hasVlans(parsedVlans.Get("br1"), 4094)

						eth1Vlans := parsedVlans.Get("eth1").Array()
						Expect(eth1Vlans).To(BeEmpty())

						br2 := parsedVlans.Get("br2")
						hasVlans(br2, 1)

						eth2Vlans := parsedVlans.Get("eth2")
						hasVlans(eth2Vlans, 1)
					}
					return true
				}).Should(BeTrue())
			})
		})
	})
})
