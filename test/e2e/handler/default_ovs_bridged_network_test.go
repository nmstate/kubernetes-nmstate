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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func ovsBridgeWithTheDefaultInterface(ovsBridgeName string, defaultInterfaceMac string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: ovs0
  type: ovs-interface
  state: up
  ipv4:
    enabled: true
    dhcp: true
  mac-address: %s
- name: %s
  type: ovs-bridge
  state: up
  bridge:
    options:
      stp: true
    port:
    - name: %s
    - name: ovs0
`, defaultInterfaceMac, ovsBridgeName, primaryNic))
}

func ovsBridgeWithTheDefaultInterfaceAbsent(ovsBridgeName string, ovsBridgeInternalPortName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    enabled: true
    dhcp: true
- name: %s
  type: ovs-interface
  state: absent
- name: %s
  type: ovs-bridge
  state: absent
`, primaryNic, ovsBridgeInternalPortName, ovsBridgeName))
}

var _ = Describe("NodeNetworkConfigurationPolicy default ovs-bridged network", func() {
	Context("when there is a default interface with dynamic address", func() {
		const (
			ovsDefaultNetwork     = "ovs-default-network"
			ovsBridgeInternalPort = "ovs0"
		)
		var (
			node     string
			ipv4Addr = ""
			macAddr  = ""
		)

		BeforeEach(func() {
			node = nodes[0]

			Byf("Check %s is the default route interface and has dynamic address", primaryNic)
			defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
			Expect(dhcpFlag(node, primaryNic)).Should(BeTrue())

			By("Fetching node current IP address and MAC")
			Eventually(func() string {
				ipv4Addr = ipv4Address(node, primaryNic)
				return ipv4Addr
			}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), fmt.Sprintf("Interface %s has no ipv4 address", primaryNic))
			macAddr = macAddress(node, primaryNic)
		})

		PContext(
			"and ovs bridge on top of the default interface BZ:[https://bugzilla.redhat.com/show_bug.cgi?id=2011879,"+
				"https://bugzilla.redhat.com/show_bug.cgi?id=2012420]",
			func() {
				BeforeEach(func() {
					Byf("Creating the %s policy", ovsDefaultNetwork)
					setDesiredStateWithPolicyAndNodeSelectorEventually(
						ovsDefaultNetwork, ovsBridgeWithTheDefaultInterface(bridge1, macAddr),
						map[string]string{"kubernetes.io/hostname": node},
					)

					By("Waiting until the node becomes ready again")
					waitForNodesReady()

					By("Waiting for policy to be ready")
					waitForAvailablePolicy(ovsDefaultNetwork)
				})

				AfterEach(func() {
					Byf("Removing bridge and configuring %s with dhcp", primaryNic)
					setDesiredStateWithPolicy(ovsDefaultNetwork, ovsBridgeWithTheDefaultInterfaceAbsent(bridge1, ovsBridgeInternalPort))

					By("Waiting until the node becomes ready again")
					waitForNodesReady()

					By("Waiting for policy to be ready")
					waitForAvailablePolicy(ovsDefaultNetwork)

					Byf("Check %s has the default ip address", primaryNic)
					Eventually(func() string {
						return ipv4Address(node, primaryNic)
					}, 30*time.Second, 1*time.Second).Should(Equal(ipv4Addr), fmt.Sprintf("Interface %s address is not the original one", primaryNic))

					Byf("Check %s is back as the default route interface", primaryNic)
					defaultRouteNextHopInterface(node).Should(Equal(primaryNic))

					Byf("Remove the %s policy", ovsDefaultNetwork)
					deletePolicy(ovsDefaultNetwork)

					By("Reset desired state at all nodes")
					resetDesiredStateForNodes()
				})

				checkThatOvsBridgeTookOverTheDefaultIP := func(node string, internalPortName string) {
					By("Verifying that ovs-interface obtained node's default IP")
					Eventually(
						func() string {
							return ipv4Address(node, internalPortName)
						},
						15*time.Second,
						1*time.Second,
					).Should(Equal(ipv4Addr), fmt.Sprintf("Interface %s has not taken over the %s address", bridge1, primaryNic))

					By("Verify that next-hop-interface for default route is ovs0")
					defaultRouteNextHopInterface(node).Should(Equal(ovsBridgeInternalPort))
				}

				It("should successfully move default IP address to the ovs-interface", func() {
					checkThatOvsBridgeTookOverTheDefaultIP(node, ovsBridgeInternalPort)
				})

				It("should keep the default IP address after node reboot", func() {
					err := restartNode(node)
					Expect(err).ToNot(HaveOccurred())

					By("Wait for policy re-reconciled after node reboot")
					waitForPolicyTransitionUpdate(ovsDefaultNetwork)
					waitForAvailablePolicy(ovsDefaultNetwork)

					Byf("Node %s was rebooted, verifying that bridge took over the default IP", node)
					checkThatOvsBridgeTookOverTheDefaultIP(node, ovsBridgeInternalPort)
				})
			},
		)

		Context("when desiredState is configured with internal port with wrong IP address", func() {
			const (
				ovsWrongIPPolicy    = "ovs-wrong-ip"
				ovsInternalPortName = "ovs666"
			)
			ovsBridgeWithInternalPortAndWrongIP := func(bridgeName string, internalPortName string, internalPortMac string) nmstate.State {
				return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-interface
    state: up
    mac-address: %s
    ipv4:
      enabled: true
      dhcp: false
      address:
        - ip: 1.2.3.4
          prefix-length: 24
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: false
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: true
      port:
        - name: %s
        - name: %s`,
					internalPortName, internalPortMac, primaryNic, bridgeName, primaryNic, internalPortName))
			}

			BeforeEach(func() {
				node = nodes[0]

				Byf("Check %s is the default route interface and has dynamic address", primaryNic)
				defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
				Expect(dhcpFlag(node, primaryNic)).Should(BeTrue())

				By("Fetching node current IP address and MAC")
				Eventually(func() string {
					ipv4Addr = ipv4Address(node, primaryNic)
					return ipv4Addr
				}, 15*time.Second, 1*time.Second).ShouldNot(BeEmpty(), fmt.Sprintf("Interface %s has no ipv4 address", primaryNic))
				macAddr = macAddress(node, primaryNic)
			})

			AfterEach(func() {
				Byf("Remove the %s policy", ovsWrongIPPolicy)
				deletePolicy(ovsWrongIPPolicy)

				By("Reset desired state at all nodes")
				resetDesiredStateForNodes()
			})

			It("should fail to configure and rollback", func() {
				Byf("Creating the %s policy", ovsWrongIPPolicy)
				setDesiredStateWithPolicyAndNodeSelectorEventually(
					ovsWrongIPPolicy, ovsBridgeWithInternalPortAndWrongIP(bridge1, ovsInternalPortName, macAddr),
					map[string]string{"kubernetes.io/hostname": node},
				)
				By("Wait for the policy to fail")
				waitForDegradedPolicy(ovsWrongIPPolicy)

				Byf("Check %s still has the default ip address", primaryNic)
				Eventually(func() string {
					return ipv4Address(node, primaryNic)
				}, 30*time.Second, 1*time.Second).Should(Equal(ipv4Addr), fmt.Sprintf("Interface %s address is not the original one", primaryNic))

				Byf("Check %s is still the default route interface", primaryNic)
				defaultRouteNextHopInterface(node).Should(Equal(primaryNic))
			})
		})
	})
})
