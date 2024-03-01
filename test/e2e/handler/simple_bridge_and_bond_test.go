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

package handler

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstateapiv2 "github.com/nmstate/nmstate/rust/src/go/api/v2"
)

func bondAbsent(bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: absent
`, bondName))
}

func brAndBondAbsent(bridgeName, bondName string) nmstate.State {
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
      %s:
        - %s
      options:
        miimon: %s
`, bondName, portFieldName, firstSecondaryNic, fmt.Sprintf(miimonFormat, 120)))
}

func brWithBondUp(bridgeName, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      %s:
        - %s
      options:
        miimon: %s
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: %s
`, bondName, portFieldName, firstSecondaryNic, fmt.Sprintf(miimonFormat, 120), bridgeName, bondName))
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
      miimon: %s
    %s:
    - %s
    - %s
`, bondName, fmt.Sprintf(miimonFormat, 140), portFieldName, firstSecondaryNic, secondSecondaryNic))
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
      miimon: %s
    %s:
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
`, bondName, fmt.Sprintf(miimonFormat, 140), portFieldName, firstSecondaryNic, secondSecondaryNic, bondName, bondName))
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
			It("should have the linux bridge at currentState with vlan_filtering 0", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 0"))
				}
			})
		})
		Context("with a linux bridge up with a port with disabled VLAN", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(linuxBrUpWithDisabledVlan(bridge1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState with vlan_filtering 0 and no default vlan range configured", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 0"))

					getVLANFlagsEventually(node, firstSecondaryNic, 1).
						Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					vlansCardinality(node, firstSecondaryNic).Should(Equal(0))
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
			It("should have the linux bridge at currentState with vlan_filtering 1", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 1"))
				}
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					getVLANFlagsEventually(node, bridge1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					getVLANFlagsEventually(node, firstSecondaryNic, 1).
						Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, firstSecondaryNic, 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, secondSecondaryNic, 1).
						Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, secondSecondaryNic, 2, 4094).Should(Succeed())
				}
			})
			Context("and vlan field reset at ports", func() {
				BeforeEach(func() {
					updateDesiredStateAndWait(linuxBrUpWithDisabledVlan(bridge1))
				})
				AfterEach(func() {
					updateDesiredStateAndWait(linuxBrAbsent(bridge1))
					for _, node := range nodes {
						interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
					}
					resetDesiredStateForNodes()
				})
				It("should have the linux bridge at currentState with vlan_filtering 0 and no default vlan range configured", func() {
					Skip("Pending on https://bugzilla.redhat.com/show_bug.cgi?id=2067058 land centos stream 8")
					for _, node := range nodes {
						interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
						bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 0"))

						getVLANFlagsEventually(node, firstSecondaryNic, 1).
							Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
						vlansCardinality(node, firstSecondaryNic).Should(Equal(0))
					}
				})
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

				expectedNetworkState := nmstateapiv2.NetworkState{}
				Expect(yaml.Unmarshal(bondUp(bond1).Raw, &expectedNetworkState)).To(Succeed())
				expectedBond := expectedNetworkState.Interfaces[0]

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
				expectedNetworkState := nmstateapiv2.NetworkState{}
				Expect(yaml.Unmarshal(brWithBondUp(bridge1, bond1).Raw, &expectedNetworkState)).To(Succeed())

				filterInPortName := func(ports *[]nmstateapiv2.BridgePortConfig) []nmstateapiv2.BridgePortConfig {
					if ports == nil {
						return nil
					}
					filteredPorts := []nmstateapiv2.BridgePortConfig{}
					for _, p := range *ports {
						filteredPorts = append(filteredPorts, nmstateapiv2.BridgePortConfig{
							BridgePortConfigMetaData: nmstateapiv2.BridgePortConfigMetaData{
								Name: p.Name,
							},
						})
					}
					return filteredPorts
				}
				expectedBond := findInterfaceByName(bond1, expectedNetworkState.Interfaces)
				expectedBridge := findInterfaceByName(bridge1, expectedNetworkState.Interfaces)
				obtainedBridge := &nmstateapiv2.Interface{}
				for _, node := range nodes {
					interfacesForNode(node).Should(SatisfyAll(
						ContainElement(matchingBond(*expectedBond)),
						ContainElement(HaveField("Name", expectedBridge.Name), obtainedBridge),
					))

					getVLANFlagsEventually(node, bridge1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					hasVlans(node, bond1, 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, bond1, 1).Should(ConsistOf("PVID", Or(Equal("Egress Untagged"), Equal("untagged"))))
					vlansCardinality(node, firstSecondaryNic).Should(Equal(0))
					vlansCardinality(node, secondSecondaryNic).Should(Equal(0))
				}
				Expect(obtainedBridge.Type).To(Equal(expectedBridge.Type))
				Expect(obtainedBridge.State).To(Equal(expectedBridge.State))
				Expect(obtainedBridge.BridgeInterface).ToNot(BeNil())
				Expect(obtainedBridge.BridgeInterface.BridgeConfig).ToNot(BeNil())
				Expect(obtainedBridge.BridgeConfig.Ports).To(WithTransform(filterInPortName, ConsistOf(*expectedBridge.BridgeConfig.Ports)))
			})
		})
		Context("with bond interface that has 2 eths as ports", func() {
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
			It("should have the bond interface with 2 ports at currentState", func() {
				expectedNetworkState := nmstateapiv2.NetworkState{}
				Expect(yaml.Unmarshal(bondUpWithEth1AndEth2(bond1).Raw, &expectedNetworkState)).To(Succeed())
				expectedBond := findInterfaceByName(bond1, expectedNetworkState.Interfaces)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(matchingBond(*expectedBond)))
				}
			})
		})
		Context("with bond interface that has 2 eths as ports and vlan tag on the bond", func() {
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
			It("should have the bond interface with 2 ports at currentState", func() {
				expectedNetworkState := nmstateapiv2.NetworkState{}
				Expect(yaml.Unmarshal(bondUpWithEth1Eth2AndVlan(bond1).Raw, &expectedNetworkState)).To(Succeed())
				expectedBond := findInterfaceByName(bond1, expectedNetworkState.Interfaces)
				expectedVlanBond102 := findInterfaceByName(fmt.Sprintf("%s.102", bond1), expectedNetworkState.Interfaces)

				for _, node := range nodes {
					interfacesForNode(node).WithTimeout(5 * time.Second).WithPolling(time.Second).Should(SatisfyAll(
						ContainElement(matchingBond(*expectedBond)),
						ContainElement(SatisfyAll(
							HaveField("Name", expectedVlanBond102.Name),
							HaveField("Type", expectedVlanBond102.Type),
							HaveField("State", expectedVlanBond102.State)))))
				}
			})
		})
	})
})
