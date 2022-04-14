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

package upgrade

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/operator"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

type upgradePolicyCase struct {
	name         string
	fileName     string
	policyName   string
	cleanupState *nmstate.State
	ifaceNames   []string
}

var _ = Describe("Upgrade", func() {
	interfaceAbsent := func(iface string) nmstate.State {
		return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
  state: absent
`, iface))
	}

	cleanDNSDesiredState := nmstate.NewState(`dns-resolver:
  config:
    search: []
    server: []
interfaces:
- name: eth1
	state: absent
`)

	kubectlAndCheck := func(command ...string) {
		out, err := cmd.Kubectl(command...)
		Expect(err).ShouldNot(HaveOccurred(), out)
	}

	examples := []upgradePolicyCase{
		{
			name:       "Ethernet",
			fileName:   "ethernet.yaml",
			policyName: "ethernet",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Linux bridge",
			fileName:   "linux-bridge.yaml",
			policyName: "linux-bridge",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "Linux bridge with custom vlan",
			fileName:   "linux-bridge-vlan.yaml",
			policyName: "linux-bridge-vlan",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "OVS bridge with interface",
			fileName:   "ovs-bridge-iface.yaml",
			policyName: "ovs-bridge-iface",
			ifaceNames: []string{"ovs0", "ovs-bridge", "eth1"},
		},
		{
			name:       "Linux bonding",
			fileName:   "bond.yaml",
			policyName: "bond",
			ifaceNames: []string{"bond0"},
		},
		{
			name:       "Linux bonding and VLAN",
			fileName:   "bond-vlan.yaml",
			policyName: "bond-vlan",
			ifaceNames: []string{"bond0.102", "bond0"},
		},
		{
			name:       "VLAN",
			fileName:   "vlan.yaml",
			policyName: "vlan",
			ifaceNames: []string{"eth1.102", "eth1"},
		},
		{
			name:       "DHCP",
			fileName:   "dhcp.yaml",
			policyName: "dhcp",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Static IP",
			fileName:   "static-ip.yaml",
			policyName: "static-ip",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Route",
			fileName:   "route.yaml",
			policyName: "route",
			ifaceNames: []string{"eth1"},
		},
		{
			name:         "DNS",
			fileName:     "dns.yaml",
			policyName:   "dns",
			ifaceNames:   []string{},
			cleanupState: &cleanDNSDesiredState,
		},
		{
			name:       "Worker selector",
			fileName:   "worker-selector.yaml",
			policyName: "worker-selector",
			ifaceNames: []string{"eth1"},
		},
	}

	createUpgradeCasePolicy := func(example upgradePolicyCase) {
		By(fmt.Sprintf("Creating policy %s", example.policyName))
		kubectlAndCheck("apply", "-f", fmt.Sprintf("test/e2e/upgrade/examples/%s", example.fileName))
		By("Waiting for policy to be available")
		kubectlAndCheck("wait", "nncp", example.policyName, "--for", "condition=Available", "--timeout", "3m")
	}

	createUpgradeCaseCleanupPolicy := func(example upgradePolicyCase) {
		if example.cleanupState != nil {
			setDesiredStateWithPolicyEventually(example.policyName, *example.cleanupState)
		}
		if len(example.ifaceNames) > 0 {
			for _, ifaceName := range example.ifaceNames {
				setDesiredStateWithPolicyEventually(
					example.policyName,
					interfaceAbsent(ifaceName),
				)
			}
		}

		kubectlAndCheck("wait", "nncp", example.policyName, "--for", "condition=Available", "--timeout", "3m")
	}

	BeforeEach(func() {
		operator.UninstallOperator(latestOperator)
		operator.InstallOperator(previousReleaseOperator)
		operator.EventuallyOperandIsReady(previousReleaseOperator)
	})

	Context("With examples", func() {
		for _, e := range examples {
			example := e

			Context(example.name, func() {
				It("should succeed applying the policy", func() {
					//TODO: remove when no longer required
					for _, policyToSkip := range []string{"vlan", "linux-bridge-vlan", "dns"} {
						if policyToSkip == example.policyName {
							Skip("Skipping due to malformed example manifest")
						}
					}
					createUpgradeCasePolicy(example)
				})
				AfterEach(func() {
					policiesLastHeartbeatTimestamps := map[string]time.Time{}

					nncps := nmstatev1.NodeNetworkConfigurationPolicyList{}
					err := testenv.Client.List(context.TODO(), &nncps)
					Expect(err).ToNot(HaveOccurred())

					By("Collecting LastHeartbeatTime timestamps of present policies")
					for _, nncp := range nncps.Items {
						availableCondition := nncp.Status.Conditions.Find(nmstate.NodeNetworkConfigurationPolicyConditionAvailable)
						Expect(availableCondition).ToNot(BeNil())
						policiesLastHeartbeatTimestamps[nncp.Name] = availableCondition.LastHeartbeatTime.Time
					}

					By("Applying new nmstate operator")
					operator.UninstallOperator(previousReleaseOperator)
					operator.InstallOperator(latestOperator)
					operator.EventuallyOperandIsReady(latestOperator)

					By("Waiting for all policies to be re-reconciled")
					allPoliciesReReconciled := func() error {
						nncps = nmstatev1.NodeNetworkConfigurationPolicyList{}
						err = testenv.Client.List(context.TODO(), &nncps)
						if err != nil {
							return err
						}
						for _, nncp := range nncps.Items {
							availableCondition := nncp.Status.Conditions.Find(nmstate.NodeNetworkConfigurationPolicyConditionAvailable)
							if availableCondition.Status != corev1.ConditionTrue {
								return fmt.Errorf("policy %s is not Available", nncp.Name)
							}
							if !availableCondition.LastHeartbeatTime.Time.After(policiesLastHeartbeatTimestamps[nncp.Name]) {
								return fmt.Errorf("policy  %s hasn't re-reconciled yet", nncp.Name)
							}
						}
						return nil
					}
					Eventually(func() error {
						return allPoliciesReReconciled()
					}, ReadTimeout, ReadInterval).Should(Succeed())

					By("Wait for policy to be Available again")
					kubectlAndCheck("wait", "nncp", example.policyName, "--for", "condition=Available", "--timeout", "3m")

					By("Apply cleanup policy configuration")
					createUpgradeCaseCleanupPolicy(example)

					By("Delete policy")
					deletePolicy(example.policyName)
				})
			})
		}
	})
})
