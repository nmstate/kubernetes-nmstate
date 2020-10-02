package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func bondAbsent(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: absent
`, bondName))
}

func brAndBondAbsent(bridgeName string, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
  - name: %s
    type: bond
    state: absent
`, bridgeName, bondName))
}

func bondUp(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - %s
      options:
        miimon: '120'
`, bondName, firstSecondaryNic))
}

func brWithBondUp(bridgeName string, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - %s
      options:
        miimon: '120'
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: %s
`, bondName, firstSecondaryNic, bridgeName, bondName))
}

func bondUpWithEth1AndEth2(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
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
    - %s
    - %s
`, bondName, firstSecondaryNic, secondSecondaryNic))
}

func bondUpWithEth1Eth2AndVlan(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
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
    - %s
    - %s
- name: %s.102
  type: vlan
  state: up
  ipv4:
    address:
    - ip: 10.102.10.10
      prefix-length: 24
    enabled: true
  vlan:
    base-iface: %s
    id: 102
`, bondName, firstSecondaryNic, secondSecondaryNic, bondName, bondName))
}

var _ = Describe("NodeNetworkState", func() {
	Context("when desiredState is configured", func() {
		Context("with a linux bridge up with no ports", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(linuxBrUpNoPorts(bridge1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState with vlan_filtering 1", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 1"))
				}
			})
		})
		Context("with a linux bridge up", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(linuxBrUp(bridge1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					getVLANFlagsEventually(node, bridge1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					getVLANFlagsEventually(node, firstSecondaryNic, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, firstSecondaryNic, 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, secondSecondaryNic, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, secondSecondaryNic, 2, 4094).Should(Succeed())
				}
			})
		})
		Context("with a active-backup miimon 100 bond interface up", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(bondUp(bond1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(bondAbsent(bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond interface at currentState", func() {
				var (
					expectedBond = interfaceByName(interfaces(bondUp(bond1)), bond1)
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(matchingBond(expectedBond)))
				}
			})
		})
		Context("with the bond interface as linux bridge port", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(brWithBondUp(bridge1, bond1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(brAndBondAbsent(bridge1, bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond in the linux bridge as port at currentState", func() {
				var (
					expectedInterfaces = interfaces(brWithBondUp(bridge1, bond1))
					expectedBond       = interfaceByName(expectedInterfaces, bond1)
					expectedBridge     = interfaceByName(expectedInterfaces, bridge1)
				)
				for _, node := range nodes {
					interfacesForNode(node).Should(SatisfyAll(
						ContainElement(matchingBond(expectedBond)),
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedBridge["name"]),
							HaveKeyWithValue("type", expectedBridge["type"]),
							HaveKeyWithValue("state", expectedBridge["state"]),
							HaveKeyWithValue("bridge", HaveKeyWithValue("port",
								ContainElement(HaveKeyWithValue("name", bond1)))),
						))))

					getVLANFlagsEventually(node, bridge1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, bond1, 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, bond1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					vlansCardinality(node, firstSecondaryNic).Should(Equal(0))
					vlansCardinality(node, secondSecondaryNic).Should(Equal(0))
				}
			})
		})
		Context("with bond interface that has 2 eths as slaves", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(bondUpWithEth1AndEth2(bond1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(bondAbsent(bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond interface with 2 slaves at currentState", func() {
				var (
					expectedBond = interfaceByName(interfaces(bondUpWithEth1AndEth2(bond1)), bond1)
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(matchingBond(expectedBond)))
				}
			})
		})
		Context("with bond interface that has 2 eths as slaves and vlan tag on the bond", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(bondUpWithEth1Eth2AndVlan(bond1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(bondAbsent(bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond interface with 2 slaves at currentState", func() {
				var (
					expectedBond        = interfaceByName(interfaces(bondUpWithEth1Eth2AndVlan(bond1)), bond1)
					expectedVlanBond102 = interfaceByName(interfaces(bondUpWithEth1Eth2AndVlan(bond1)), fmt.Sprintf("%s.102", bond1))
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(SatisfyAll(
						ContainElement(matchingBond(expectedBond)),
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedVlanBond102["name"]),
							HaveKeyWithValue("type", expectedVlanBond102["type"]),
							HaveKeyWithValue("state", expectedVlanBond102["state"])))))
				}
			})
		})
	})
})
