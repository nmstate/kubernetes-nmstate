package helper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var (
	badYaml = nmstate.NewState("}")
	empty   = nmstate.NewState("")

	noBridges = nmstate.NewState(`interfaces:
  - name: bond1
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      port:
        - eth1
      options:
        miimon: '120'
`)
	noBridgesUp = nmstate.NewState(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br1
    type: linux-bridge
    state: down
  - name: br2
    type: linux-bridge
    state: absent
`)

	bridgeWithNoPorts = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
`)

	someBridgesUp = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth1
  - name: br2
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth2
        - name: eth3
  - name: br3
    type: linux-bridge
    state: down
  - name: br4
    type: linux-bridge
    state: absent
`)
	expectedSomeBridgesUpDefaults = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth1
        vlan:
          mode: trunk
          trunk-tags:
          - id-range:
              max: 4094
              min: 2
  - name: br2
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth2
        vlan:
          mode: trunk
          trunk-tags:
          - id-range:
              max: 4094
              min: 2
      - name: eth3
        vlan:
          mode: trunk
          trunk-tags:
          - id-range:
              max: 4094
              min: 2
  - name: br3
    type: linux-bridge
    state: down
  - name: br4
    type: linux-bridge
    state: absent
`)
	bridgeWithCustomVlan = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth1
        vlan:
          mode: trunk
          trunk-tags:
          - id-range:
              max: 200
              min: 2
          - id: 101
          tag: 100
          enable-native: true
`)
	bridgeWithDisabledVlan = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth1
        vlan: {}
`)
	someBridgesWithVlanConfiguration = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth1
  - name: br2
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth2
          vlan:
            mode: trunk
            trunk-tags:
            - id: 101
            - id: 102
            tag: 100
            enable-native: true
        - name: eth3
  - name: br3
    type: linux-bridge
    state: down
  - name: br4
    type: linux-bridge
    state: absent
`)
	expectedSomeBridgesWithVlanConfigurationDefaults = nmstate.NewState(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth1
          vlan:
            mode: trunk
            trunk-tags:
            - id-range:
                max: 4094
                min: 2
  - name: br2
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth2
          vlan:
            mode: trunk
            trunk-tags:
            - id: 101
            - id: 102
            tag: 100
            enable-native: true
        - name: eth3
          vlan:
            mode: trunk
            trunk-tags:
            - id-range:
                max: 4094
                min: 2
  - name: br3
    type: linux-bridge
    state: down
  - name: br4
    type: linux-bridge
    state: absent
`)
)

var _ = Describe("Network desired state bridge parser", func() {
	var (
		updatedDesiredState nmstate.State
		desiredState        nmstate.State
		err                 error
	)
	JustBeforeEach(func() {
		updatedDesiredState, err = ApplyDefaultVlanFiltering(desiredState)
	})
	Context("when desired state is not a yaml", func() {
		BeforeEach(func() {
			desiredState = badYaml
		})
		It("should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when desired state is empty", func() {
		BeforeEach(func() {
			desiredState = empty
		})
		It("should not be changed", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDesiredState).To(MatchYAML(desiredState))
		})
	})
	Context("when there are no bridges", func() {
		BeforeEach(func() {
			desiredState = noBridges
		})
		It("should not be changed", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDesiredState).To(MatchYAML(desiredState))
		})
	})
	Context("when there are no bridges up", func() {
		BeforeEach(func() {
			desiredState = noBridgesUp
		})
		It("should not be changed", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDesiredState).To(MatchYAML(desiredState))
		})
	})
	Context("when there are no ports in the bridge", func() {
		BeforeEach(func() {
			desiredState = bridgeWithNoPorts
		})
		It("should not be changed", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDesiredState).To(MatchYAML(desiredState))
		})
	})
	Context("when there are bridges up", func() {
		BeforeEach(func() {
			desiredState = someBridgesUp
		})
		It("should add default vlan filtering to linux-bridge ports", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDesiredState).To(MatchYAML(expectedSomeBridgesUpDefaults))
		})
		Context("when there is custom vlan configuration on linux-bridge port", func() {
			BeforeEach(func() {
				desiredState = bridgeWithCustomVlan
			})
			It("should keep custom vlan configuration intact", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedDesiredState).To(MatchYAML(desiredState))
			})
		})
		Context("when there is empty vlan configuration", func() {
			BeforeEach(func() {
				desiredState = bridgeWithDisabledVlan
			})
			It("should keep custom vlan configuration intact", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedDesiredState).To(MatchYAML(desiredState))
			})
		})
		Context("when some ports have vlan configuration while other do not", func() {
			BeforeEach(func() {
				desiredState = someBridgesWithVlanConfiguration
			})
			It("should keep custom vlan configuration intact", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedDesiredState).To(MatchYAML(expectedSomeBridgesWithVlanConfigurationDefaults))
			})
		})
	})
})

var _ = Describe("test listing linux bridges with ports", func() {
	currentState := nmstate.NewState(`interfaces:
  - name: br22
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: test-veth1
        vlan:
          enable-native: true
  - name: br3
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth2
      - name: eth3
  - name: br4
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth4
  - name: br5
    type: linux-bridge
    state: up
    bridge:
      port: []
  - name: br6
    type: linux-bridge
    state: down
    bridge:
      port:
      - name: eth666
  - name: br7
    type: linux-bridge
    state: absent
    bridge:
      port:
      - name: eth777
`)
	It("should list active bridges with at least one port", func() {
		upBridgesWithPorts, err := GetUpLinuxBridgesWithPorts(currentState)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(upBridgesWithPorts).To(
			SatisfyAll(
				HaveKeyWithValue("br3", []string{"eth2", "eth3"}),
				HaveKeyWithValue("br4", []string{"eth4"}),
				Not(HaveKey("br22")),
				Not(HaveKey("br5")),
				Not(HaveKey("br6")),
				Not(HaveKey("br7")),
			))
	})
})
