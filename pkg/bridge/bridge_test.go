/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bridge

import (
	. "github.com/onsi/ginkgo/v2"
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
