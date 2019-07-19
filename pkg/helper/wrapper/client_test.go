package wrapper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	empty                = nmstatev1alpha1.State("")
	withoutVlanFiltering = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br0
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
	vlanFiltering = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br0
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
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        vlan-filtering: true
        stp:
          enabled: false
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)
	vlanFilteringAndBridgeVlans = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br0
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
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        vlan-filtering: true
        stp:
          enabled: false
        vlans:
          - vlan-range-min: 10
            vlan-range-max: 15
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)
	vlanFilteringAndPortsVlans = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br0
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
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        vlan-filtering: true
        stp:
          enabled: false
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
          vlans:
            - vlan-range-min: 20
            - vlan-range-min: 200
              vlan-range-max: 205
`)
)

var _ = Describe("linux bridge vlan filterng", func() {
	var (
		desiredState   nmstatev1alpha1.State
		obtainedResult FilteredStateWithExtraCommands
		obtainedError  error
	)
	JustBeforeEach(func() {
		obtainedResult, obtainedError = processLinuxBridgeVlans(desiredState)
	})
	Context("when desiredState is empty", func() {
		BeforeEach(func() {
			desiredState = empty
		})
		It("should have empty desiredState", func() {
			Expect(obtainedResult.state).To(Equal(empty))
		})
		It("should not return any extra command", func() {
			Expect(obtainedResult.extraCommands).To(BeEmpty())
		})
		It("should not error", func() {
			Expect(obtainedError).ToNot(HaveOccurred())
		})
	})
	Context("when there is no vlan configuration", func() {
		BeforeEach(func() {
			desiredState = withoutVlanFiltering
		})
		It("should not modify state", func() {
			Expect(obtainedResult.state).To(MatchYAML(withoutVlanFiltering))
		})
		It("should not return any extra command", func() {
			Expect(obtainedResult.extraCommands).To(BeEmpty())
		})
		It("should not error", func() {
			Expect(obtainedError).ToNot(HaveOccurred())
		})
	})
	Context("when there is vlan-filtering but not vlans", func() {
		BeforeEach(func() {
			desiredState = vlanFiltering
		})
		It("should remove all vlans reference from desired state", func() {
			Expect(obtainedResult.state).To(MatchYAML(withoutVlanFiltering))
		})
		It("should set vlan_filtering to bridge", func() {
			Expect(obtainedResult.extraCommands).To(ConsistOf([]string{
				"ip link set br1 type bridge vlan_filtering 1",
			}))
		})
		It("should not error", func() {
			Expect(obtainedError).ToNot(HaveOccurred())
		})
	})
	Context("when there is vlan-filtering and bridge vlans", func() {
		BeforeEach(func() {
			desiredState = vlanFilteringAndBridgeVlans
		})
		It("should remove all vlans reference from desired state", func() {
			Expect(obtainedResult.state).To(MatchYAML(withoutVlanFiltering))
		})
		It("should set vlan_filtering to bridge", func() {
			Expect(obtainedResult.extraCommands).To(ConsistOf(
				"ip link set br1 type bridge vlan_filtering 1",
				"bridge vlan add dev br1 vid 10 self",
				"bridge vlan add dev br1 vid 11 self",
				"bridge vlan add dev br1 vid 12 self",
				"bridge vlan add dev br1 vid 13 self",
				"bridge vlan add dev br1 vid 14 self",
				"bridge vlan add dev br1 vid 15 self",
			))
		})
		It("should not error", func() {
			Expect(obtainedError).ToNot(HaveOccurred())
		})
	})
	Context("when there is vlan-filtering and ports vlans", func() {
		BeforeEach(func() {
			desiredState = vlanFilteringAndPortsVlans
		})
		It("should remove all vlans reference from desired state", func() {
			Expect(obtainedResult.state).To(MatchYAML(withoutVlanFiltering))
		})
		It("should set vlan_filtering to bridge", func() {
			Expect(obtainedResult.extraCommands).To(ConsistOf(
				"ip link set br1 type bridge vlan_filtering 1",
				"bridge vlan add dev eth1 vid 20",
				"bridge vlan add dev eth1 vid 200",
				"bridge vlan add dev eth1 vid 201",
				"bridge vlan add dev eth1 vid 202",
				"bridge vlan add dev eth1 vid 203",
				"bridge vlan add dev eth1 vid 204",
				"bridge vlan add dev eth1 vid 205",
			))
		})
		It("should not error", func() {
			Expect(obtainedError).ToNot(HaveOccurred())
		})
	})

})
