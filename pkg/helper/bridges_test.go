package helper

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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
      slaves:
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
      - name: eth4
      - name: eth5
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
      port:
      - name: eth5
  - name: br7
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth777
  - name: br8
    type: linux-bridge
    state: up
    bridge:
      port:
      - name: eth888
`)
	desiredState := nmstate.NewState(`interfaces:
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
  state: up
  bridge:
    port:
    - name: eth777
    - name: eth778
    - name: eth779
- name: br8
  type: linux-bridge
  state: absent
  bridge:
    port:
    - name: eth888
`)
	expectedFilteredExistingBridgesWithPorts := map[string][]string{
		"br3": {"eth2", "eth3"},
		"br4": {"eth4"},
		"br7": {"eth777"},
	}
	It("should filter correct bridges and ports", func() {
		upBridgesWithPortsAtCurrentState, err := GetUpLinuxBridgesWithPorts(currentState)
		Expect(err).ShouldNot(HaveOccurred())

		filteredExistingUpBridgesWithPorts, err := filterExistingLinuxBridgesWithPorts(upBridgesWithPortsAtCurrentState, desiredState)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(filteredExistingUpBridgesWithPorts).To(Equal(expectedFilteredExistingBridgesWithPorts))
	})
})

var _ = Describe("testing slice intersection", func() {
	type intersectionCase struct {
		s1                   []string
		s2                   []string
		expectedIntersection []string
	}

	table.DescribeTable("Slice intersection cases",
		func(c intersectionCase) {
			result := intersectSlices(c.s1, c.s2)
			Expect(result).To(Equal(c.expectedIntersection))
		},
		table.Entry(
			"Both slices empty",
			intersectionCase{
				s1:                   []string{},
				s2:                   []string{},
				expectedIntersection: []string{},
			}),
		table.Entry("Empty first slice",
			intersectionCase{
				s1:                   []string{},
				s2:                   []string{"foo"},
				expectedIntersection: []string{},
			}),
		table.Entry("Empty second slice",
			intersectionCase{
				s1:                   []string{"foo"},
				s2:                   []string{},
				expectedIntersection: []string{},
			}),
		table.Entry("No common elements",
			intersectionCase{
				s1:                   []string{"foo"},
				s2:                   []string{"bar"},
				expectedIntersection: []string{},
			}),
		table.Entry("One common element with extra in first slice",
			intersectionCase{
				s1:                   []string{"foo", "bar"},
				s2:                   []string{"bar"},
				expectedIntersection: []string{"bar"},
			}),
		table.Entry("One common element with extra in first slice",
			intersectionCase{
				s1:                   []string{"bar"},
				s2:                   []string{"bar", "foo"},
				expectedIntersection: []string{"bar"},
			}),
		table.Entry("One common element with extra in first slice",
			intersectionCase{
				s1:                   []string{"bar"},
				s2:                   []string{"bar", "foo"},
				expectedIntersection: []string{"bar"},
			}),
		table.Entry("Both identical with two elements",
			intersectionCase{
				s1:                   []string{"foo", "bar"},
				s2:                   []string{"bar", "foo"},
				expectedIntersection: []string{"bar", "foo"},
			}),
		table.Entry("Duplicates in first slice",
			intersectionCase{
				s1:                   []string{"foo", "bar", "one", "two", "three", "one", "two", "three"},
				s2:                   []string{"bar", "foo", "three", "one"},
				expectedIntersection: []string{"bar", "foo", "three", "one"},
			}),
		table.Entry("Duplicates in second slice",
			intersectionCase{
				s1:                   []string{"bar", "foo", "three", "one"},
				s2:                   []string{"foo", "bar", "one", "two", "three", "one", "two", "three"},
				expectedIntersection: []string{"foo", "bar", "one", "three"},
			}),
	)
})
