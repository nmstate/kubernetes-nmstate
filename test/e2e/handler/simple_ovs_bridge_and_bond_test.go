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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func ovsBrUpLAGEth1AndEth2(bridgeName string, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port:
        - name: %s
          link-aggregation:
            mode: balance-slb
            port:
              - name: %s
              - name: %s
`, bridgeName, bondName, firstSecondaryNic, secondSecondaryNic))
}

func ovsBrUpLAGEth1Eth2WithInternalPort(bridgeName string, internalPortName string, internalPortMac string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-interface
    state: up
    mac-address: %s
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
        mcast-snooping-enable: false
        rstp: false
      port:
      - name: bond0
        link-aggregation:
          mode: balance-slb
          port:
          - name: %s
          - name: %s
      - name: %s
`, internalPortName, internalPortMac, bridgeName, firstSecondaryNic, secondSecondaryNic, internalPortName))
}

func ovsBrUpLinuxBondEth1AndEth2(bridgeName string, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: balance-rr
      options:
        miimon: %s
      %s:
        - %s
        - %s
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port:
        - name: %s
`, bondName, fmt.Sprintf(miimonFormat, 140), portFieldName, firstSecondaryNic, secondSecondaryNic, bridgeName, bondName))
}

func ovsBrAndBondAbsent(bridgeName string, bondName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: absent
  - name: %s
    type: ovs-bridge
    state: absent
`, bondName, bridgeName))
}

func ovsBrAndInternalPortAbsent(bridgeName string, internalPortName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-interface
    state: absent
  - name: %s
    type: ovs-bridge
    state: absent
`, internalPortName, bridgeName))
}

var _ = Describe("OVS Bridge", func() {
	Context("when desiredState is updated with ovs-bridge with link aggregation port", func() {
		BeforeEach(func() {
			updateDesiredStateAndWait(ovsBrUpLAGEth1AndEth2(bridge1, bond1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(ovsBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			resetDesiredStateForNodes()
		})
		It("should have the ovs-bridge at currentState", func() {
			By("Verify all required interfaces are present at currentState")
			for _, node := range nodes {
				interfacesForNode(node).Should(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", bridge1),
					HaveKeyWithValue("type", "ovs-bridge"),
					HaveKeyWithValue("state", "up"),
				)))
			}
		})
	})
	PContext("when desiredState is updated with ovs-bridge with linux bond as port [BZ: https://bugzilla.redhat.com/show_bug.cgi?id=2005240]", func() {
		BeforeEach(func() {
			updateDesiredStateAndWait(ovsBrUpLinuxBondEth1AndEth2(bridge1, bond1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(ovsBrAndBondAbsent(bridge1, bond1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
			}
			resetDesiredStateForNodes()
		})
		It("should have the ovs-bridge and bond at currentState", func() {
			By("Verify all required interfaces are present at currentState")
			for _, node := range nodes {
				interfacesForNode(node).Should(SatisfyAll(
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", bridge1),
						HaveKeyWithValue("type", "ovs-bridge"),
						HaveKeyWithValue("state", "up"),
					)),
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", bond1),
						HaveKeyWithValue("type", "bond"),
						HaveKeyWithValue("state", "up"),
					))))
			}
		})
	})
	Context("when desiredState is updated with ovs-bridge with link aggregation port and ovs-interface port", func() {
		const ovsPortName = "ovs1"
		var (
			designatedNode string
			macAddr        = ""
		)
		BeforeEach(func() {
			designatedNode = nodes[0]

			By(fmt.Sprintf("Getting mac address of %s on %s", firstSecondaryNic, designatedNode))
			macAddr = macAddress(designatedNode, firstSecondaryNic)

			By("Creating policy with desiredState")
			updateDesiredStateAndWait(ovsBrUpLAGEth1Eth2WithInternalPort(bridge1, ovsPortName, macAddr))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(ovsBrAndInternalPortAbsent(bridge1, ovsPortName))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(ovsPortName))
			}
			resetDesiredStateForNodes()
		})

		It("should have the ovs-bridge and internal port at currentState", func() {
			By("Verify all required interfaces are present at currentState")
			interfacesForNode(designatedNode).Should(SatisfyAll(
				ContainElement(SatisfyAll(
					HaveKeyWithValue("name", bridge1),
					HaveKeyWithValue("type", "ovs-bridge"),
					HaveKeyWithValue("state", "up"),
				)),
				ContainElement(SatisfyAll(
					HaveKeyWithValue("name", ovsPortName),
					HaveKeyWithValue("type", "ovs-interface"),
					HaveKeyWithValue("state", "up"),
				))))
		})
	})
})
