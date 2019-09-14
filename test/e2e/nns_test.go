package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tidwall/gjson"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		bond1Up = nmstatev1alpha1.State(`interfaces:
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
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: eth1
        - name: eth2
`)

		br1WithBond1Up = nmstatev1alpha1.State(`interfaces:
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
`)

		br1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
  - name: eth1
    type: ethernet
    state: absent
`)
		bond0UpWithEth1AndEth2 = nmstatev1alpha1.State(`interfaces:
- name: bond0
  type: bond
  state: up
  ipv4:
    address:
    - ip: 10.10.10.10
      prefix-length: 24
    enabled: true
  link-aggregation:
    mode: balance-rr
    options:
      miimon: '140'
    slaves:
    - eth1
    - eth2
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
				deleteConnectionAtNodes("eth2")
				deleteConnectionAtNodes("br1")
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).Should(ContainElement("br1"))
				}
				Eventually(func() bool {
					for _, bridgeVlans := range bridgeVlansAtNodes() {
						hasVlans(bridgeVlans, "eth1", 2, 4094)
						hasVlans(bridgeVlans, "eth2", 2, 4094)
						hasVlans(bridgeVlans, "br1", 1, 1)
					}
					return true
				}).Should(BeTrue(), "Incorrect br1 bridge vlan ids")

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
					expectedInterfaces = interfaces(br1WithBond1Up)
					expectedBond       = interfaceByName(expectedInterfaces, "bond1")
					expectedBridge     = interfaceByName(expectedInterfaces, "br1")
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
							HaveKeyWithValue("bridge", HaveKeyWithValue("port",
								ContainElement(HaveKeyWithValue("name", "bond1")))),
						))))
				}
				Eventually(func() bool {
					for _, bridgeVlans := range bridgeVlansAtNodes() {
						parsedVlans := gjson.Parse(bridgeVlans)

						hasVlans(bridgeVlans, "bond1", 2, 4094)
						hasVlans(bridgeVlans, "br1", 1, 1)

						eth1Vlans := parsedVlans.Get("eth1").Array()
						Expect(eth1Vlans).To(BeEmpty())

						hasVlans(bridgeVlans, "br2", 1, 1)
						hasVlans(bridgeVlans, "eth2", 1, 1)
					}
					return true
				}).Should(BeTrue())
			})
		})
		Context("with bond interface that has 2 eths as slaves", func() {
			BeforeEach(func() {
				updateDesiredState(bond0UpWithEth1AndEth2)
			})
			AfterEach(func() {
				resetDesiredStateForNodes()

				// TODO: Add status conditions to ensure that
				//       it has being really reset so we can
				//       remove this ugly sleep
				time.Sleep(1 * time.Second)

				deleteConnectionAtNodes("eth1")
				deleteConnectionAtNodes("eth2")
				deleteConnectionAtNodes("bond0")
			})
			It("should have the bond interface with 2 slaves at currentState", func() {
				var (
					expectedBond  = interfaceByName(interfaces(bond0UpWithEth1AndEth2), "bond0")
					expectedSpecs = expectedBond["link-aggregation"].(map[string]interface{})
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(SatisfyAll(
						HaveKeyWithValue("name", expectedBond["name"]),
						HaveKeyWithValue("type", expectedBond["type"]),
						HaveKeyWithValue("state", expectedBond["state"]),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("mode", expectedSpecs["mode"])),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("options", expectedSpecs["options"])),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("slaves", ConsistOf([]string{"eth1", "eth2"}))),
					)))
				}
			})
		})
	})
})
