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
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/doc"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/operator"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("Upgrade", func() {
	interfaceAbsent := func(iface string) nmstate.State {
		return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
  state: absent
`, iface))
	}

	previousTagExamplesPath := "test/e2e/upgrade/examples"
	currentExamplesPath := "docs/examples"

	fileExists := func(path string) (bool, error) {
		_, err := os.Stat(path)
		exists := false

		if err == nil {
			exists = true
		} else if os.IsNotExist(err) {
			err = nil
		}
		return exists, err
	}

	kubectlAndCheck := func(command ...string) {
		out, err := cmd.Kubectl(command...)
		Expect(err).ShouldNot(HaveOccurred(), out)
	}

	createUpgradeCasePolicy := func(example doc.ExampleSpec) {
		examplePath := path.Join(previousTagExamplesPath, example.FileName)
		exists, err := fileExists(examplePath)
		Expect(err).NotTo(HaveOccurred())
		if !exists {
			examplePath = path.Join(currentExamplesPath, example.FileName)
		}

		By(fmt.Sprintf("Creating policy %s", example.PolicyName))
		kubectlAndCheck("apply", "-f", examplePath)
		By("Waiting for policy to be available")
		kubectlAndCheck("wait", "nncp", example.PolicyName, "--for", "condition=Available", "--timeout", "3m")
	}

	createUpgradeCaseCleanupPolicy := func(example doc.ExampleSpec) {
		if example.CleanupState != nil {
			setDesiredStateWithPolicyEventually(example.PolicyName, *example.CleanupState)
		}
		if len(example.IfaceNames) > 0 {
			for _, ifaceName := range example.IfaceNames {
				setDesiredStateWithPolicyEventually(
					example.PolicyName,
					interfaceAbsent(ifaceName),
				)
			}
		}

		kubectlAndCheck("wait", "nncp", example.PolicyName, "--for", "condition=Available", "--timeout", "3m")
	}

	BeforeEach(func() {
		operator.UninstallOperator(latestOperator)
		operator.InstallOperator(previousReleaseOperator)
		operator.EventuallyOperandIsReady(previousReleaseOperator)
	})

	Context("With examples", func() {
		for _, e := range doc.ExampleSpecs() {
			example := e

			Context(example.Name, func() {
				It("should succeed applying the policy", func() {
					//TODO: remove when no longer required
					for _, policyToSkip := range []string{"vlan", "linux-bridge-vlan", "dns"} {
						if policyToSkip == example.PolicyName {
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
					kubectlAndCheck("wait", "nncp", example.PolicyName, "--for", "condition=Available", "--timeout", "3m")

					By("Apply cleanup policy configuration")
					createUpgradeCaseCleanupPolicy(example)

					By("Delete policy")
					deletePolicy(example.PolicyName)
				})
			})
		}
	})
})
