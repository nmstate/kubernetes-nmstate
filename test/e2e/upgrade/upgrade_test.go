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
	"k8s.io/apimachinery/pkg/types"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/doc"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/operator"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("Upgrade", func() {
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
		Expect(waitForPolicyAvailable(example.PolicyName, PolicyAvailableTimeout)).To(Succeed())
	}

	// cleanupUpgradeCase restores the node network configuration changed by the
	// example. It's registered before creating the policy and never gated on the
	// upgrade succeeding, so leftovers from one example (e.g. bond0 enslaving
	// eth1/eth2) cannot break the examples running afterwards.
	cleanupUpgradeCase := func(example doc.ExampleSpec) {
		exists, err := policyExists(example.PolicyName)
		if err != nil {
			AbortSuite(fmt.Sprintf("Cannot check if policy %s has to be cleaned up: %v", example.PolicyName, err))
		}
		if !exists {
			return
		}

		By("Apply cleanup policy configuration")
		desiredState, err := cleanupDesiredState(example)
		if err != nil {
			AbortSuite(fmt.Sprintf("Cannot compose cleanup state for policy %s: %v", example.PolicyName, err))
		}

		// The enactments that could have applied the example state are captured
		// before the update, so a partial or transiently empty list afterwards
		// cannot be mistaken for a converged cleanup.
		expectedEnactments, err := policyEnactmentNames(example.PolicyName)
		if err != nil {
			AbortSuite(fmt.Sprintf("Cannot list the enactments of policy %s: %v", example.PolicyName, err))
		}

		if len(desiredState.Raw) > 0 {
			if len(expectedEnactments) == 0 {
				// Cleanup state is non-empty but no enactments exist. Only allow
				// the no-op/delete path when the policy's Ignored condition
				// confirms NoMatchingNode — any other outcome (including a
				// transient read error, or the condition not yet being set by the
				// reconciler) is treated as unsafe and aborts the suite to prevent
				// contaminated node state from reaching later examples.
				noMatchingNodeErr := retryUntil(APIRetryTimeout, func() error {
					policy := nmstatev1.NodeNetworkConfigurationPolicy{}
					if err := testenv.Client.Get(
						context.TODO(),
						types.NamespacedName{Name: example.PolicyName},
						&policy,
					); err != nil {
						return err // transient, retry
					}
					ignored := policy.Status.Conditions.Find(nmstate.NodeNetworkConfigurationPolicyConditionIgnored)
					if ignored != nil &&
						ignored.Status == corev1.ConditionTrue &&
						ignored.Reason == nmstate.NodeNetworkConfigurationPolicyConditionConfigurationNoMatchingNode {
						return nil
					}
					return fmt.Errorf("policy %s is not yet in Ignored/NoMatchingNode state", example.PolicyName)
				})
				if noMatchingNodeErr != nil {
					AbortSuite(fmt.Sprintf(
						"Policy %s has cleanup state but no enactments; NoMatchingNode not confirmed: %v",
						example.PolicyName, noMatchingNodeErr,
					))
				}
			} else {
				generation, updateErr := updatePolicyDesiredState(example.PolicyName, desiredState)
				if updateErr != nil {
					AbortSuite(fmt.Sprintf("Cannot apply cleanup state at policy %s: %v", example.PolicyName, updateErr))
				}

				By("Waiting for the cleanup configuration to be applied")
				// The policy is kept with the cleanup desired state if it is not applied,
				// so the handler can keep on reconciling it, and the suite is aborted to
				// prevent following examples from running with contaminated nodes.
				if err := waitForPolicyGenerationApplied(
					example.PolicyName, generation, expectedEnactments, PolicyAvailableTimeout,
				); err != nil {
					AbortSuite(fmt.Sprintf("Cleanup of policy %s was not applied: %v", example.PolicyName, err))
				}
			}
		}

		if err := deletePolicy(example.PolicyName); err != nil {
			AbortSuite(fmt.Sprintf("Cannot delete policy %s: %v", example.PolicyName, err))
		}
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
				// Tracks that the example policy was applied successfully, the
				// upgrade below is only meaningful (and only possible) in that case.
				policyApplied := false

				BeforeEach(func() {
					policyApplied = false
				})

				It("should succeed applying the policy", func() {
					//TODO: remove when no longer required
					for _, policyToSkip := range []string{"vlan", "linux-bridge-vlan", "dns", "enable-lldp-ethernets-up"} {
						if policyToSkip == example.PolicyName {
							Skip("Skipping due to malformed example manifest")
						}
					}
					// The cleanup is registered before creating the policy since
					// applying the manifest can create it and fail afterwards, and it
					// runs after the AfterEach nodes below even if they fail.
					DeferCleanup(func() {
						cleanupUpgradeCase(example)
					})
					createUpgradeCasePolicy(example)
					policyApplied = true
				})
				AfterEach(func() {
					if !policyApplied {
						return
					}
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
							if !availableCondition.LastHeartbeatTime.After(policiesLastHeartbeatTimestamps[nncp.Name]) {
								return fmt.Errorf("policy  %s hasn't re-reconciled yet", nncp.Name)
							}
						}
						return nil
					}
					Eventually(func() error {
						return allPoliciesReReconciled()
					}, ReReconcileTimeout, ReadInterval).Should(Succeed())

					By("Wait for policy to be Available again")
					Expect(waitForPolicyAvailable(example.PolicyName, PolicyAvailableTimeout)).To(Succeed())
				})
			})
		}
	})
})
