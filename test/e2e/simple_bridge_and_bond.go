package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		br1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
`)

		bond1Absent = nmstatev1alpha1.State(`interfaces:
  - name: bond1
    type: bond
    state: absent
`)
		br1AndBond1Absent = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
  - name: bond1
    type: bond
    state: absent
`)

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

		br1UpNoPorts = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port: []
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

		bond1UpWithEth1AndEth2 = nmstatev1alpha1.State(`interfaces:
- name: bond1
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
		Context("with a linux bridge up with no ports", func() {
			BeforeEach(func() {
				updateDesiredState(br1UpNoPorts)
			})
			AfterEach(func() {
				updateDesiredState(br1Absent)
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
				}
			})
			It("should have the linux bridge at currentState with vlan_filtering 1", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement("br1"))
					bridgeDescription(node, "br1").Should(ContainSubstring("vlan_filtering 1"))
				}
			})
		})
		Context("with a linux bridge up", func() {
			BeforeEach(func() {
				updateDesiredState(br1Up)
			})
			AfterEach(func() {
				updateDesiredState(br1Absent)
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
				}
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement("br1"))
					vlansCardinality(node, "br1").Should(Equal(0))
					hasVlans(node, "eth1", 2, 4094).Should(Succeed())
					hasVlans(node, "eth2", 2, 4094).Should(Succeed())
				}
			})
		})
		Context("with a active-backup miimon 100 bond interface up", func() {
			BeforeEach(func() {
				updateDesiredState(bond1Up)
			})
			AfterEach(func() {
				updateDesiredState(bond1Absent)
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("bond1"))
				}
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
				updateDesiredState(br1WithBond1Up)
			})
			AfterEach(func() {
				updateDesiredState(br1AndBond1Absent)
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("br1"))
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("bond1"))
				}
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

					hasVlans(node, "bond1", 2, 4094).Should(Succeed())
					vlansCardinality(node, "br1").Should(Equal(0))
					vlansCardinality(node, "eth1").Should(Equal(0))
					vlansCardinality(node, "eth2").Should(Equal(0))
				}
			})
		})
		Context("with bond interface that has 2 eths as slaves", func() {
			BeforeEach(func() {
				updateDesiredState(bond1UpWithEth1AndEth2)
			})
			AfterEach(func() {
				updateDesiredState(bond1Absent)
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement("bond1"))
				}
			})
			It("should have the bond interface with 2 slaves at currentState", func() {
				var (
					expectedBond  = interfaceByName(interfaces(bond1UpWithEth1AndEth2), "bond1")
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
