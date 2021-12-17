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
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
)

func invalidConfig(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: invalid_state
`, bridgeName))
}

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component]EnactmentCondition", func() {
	Context("when applying valid config", func() {
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))

			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("[test_id:3796]should go from Progressing to Available", func() {
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					enactmentConditionsStatusEventually(node).Should(matchConditionsFrom(enactmentconditions.SetProgressing), "should reach progressing state at %s", node)
					enactmentConditionsStatusEventually(node).Should(matchConditionsFrom(enactmentconditions.SetSuccess), "should reach available state at %s", node)
					enactmentConditionsStatusConsistently(node).Should(matchConditionsFrom(enactmentconditions.SetSuccess), "should keep available state at %s", node)
				}()
			}
			// Run the policy after we set the nnce conditions assert so we
			// make sure we catch progressing state.
			updateDesiredState(linuxBrUp(bridge1))

			wg.Wait()

			By("Check policy is at available state")
			waitForAvailableTestPolicy()
		})
	})

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			updateDesiredState(invalidConfig(bridge1))
		})

		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("[test_id:3795] should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				Byf("Check %s failing state is reached", node)
				enactmentConditionsStatusEventually(node).Should(
					SatisfyAny(
						matchConditionsFrom(enactmentconditions.SetFailedToConfigure),
						matchConditionsFrom(enactmentconditions.SetConfigurationAborted),
					), "should eventually reach failing or aborted conditions at enactments",
				)
			}
			By("Check policy is at degraded state")
			waitForDegradedTestPolicy()

			By("Check that the enactment stays in failing state")
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					Byf("Check %s failing state is kept", node)
					enactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							matchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							matchConditionsFrom(enactmentconditions.SetConfigurationAborted),
						), "should consistently keep failing or aborted conditions at enactments",
					)
				}()
			}
			wg.Wait()
		})

		It("[test_id:3795] should have up to maxUnavailable Failing and the rest Aborted ConditionType set to true", func() {
			checkEnactmentCounts := func(policy string) {
				failingConditions := 0
				abortedConditions := 0
				for _, node := range nodes {
					conditionList := enactmentConditionsStatus(node, TestPolicy)
					success, _ := matchConditionsFrom(enactmentconditions.SetFailedToConfigure).Match(conditionList)
					if success {
						failingConditions++
					}
					success, _ = matchConditionsFrom(enactmentconditions.SetConfigurationAborted).Match(conditionList)
					if success {
						abortedConditions++
					}
				}
				Expect(failingConditions).To(BeNumerically("<=", maxUnavailableNodes()), "one node only should have failing enactment")
				Expect(abortedConditions).To(Equal(len(nodes)-failingConditions), "other nodes should have aborted enactment")
			}

			By("Wait for enactments to reach failing or aborted state")
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					Byf("Check %s failing state is kept", node)
					enactmentConditionsStatusEventually(node).Should(
						SatisfyAny(
							matchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							matchConditionsFrom(enactmentconditions.SetConfigurationAborted),
						), "should consistently keep failing or aborted conditions at enactments")
				}()
			}
			wg.Wait()

			By("Check policy is at degraded state")
			waitForDegradedTestPolicy()

			By("Check that the enactments stay in failing or aborted state")
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					Byf("Check %s failing state is kept", node)
					enactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							matchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							matchConditionsFrom(enactmentconditions.SetConfigurationAborted),
						), "should consistently keep failing or aborted conditions at enactments")
				}()
			}
			wg.Wait()

			By("Check there is up to maxUnavailable failing enactments and the rest are aborted")
			checkEnactmentCounts(TestPolicy)
		})

		It("should mark policy as Degraded as soon as first enactment fails", func() {
			failingEnactmentsCount := func(policy string) int {
				failingConditions := 0
				for _, node := range nodes {
					conditionList := enactmentConditionsStatus(node, TestPolicy)
					found, _ := matchConditionsFrom(enactmentconditions.SetFailedToConfigure).Match(conditionList)
					if found {
						failingConditions++
					}
				}
				return failingConditions
			}

			By("Waiting for first enactment to fail")
			Eventually(func() int {
				return failingEnactmentsCount(TestPolicy)
			}, 180*time.Second, 1*time.Second).Should(BeNumerically(">=", 1))

			By("Checking the policy is marked as Degraded")
			Eventually(func() nmstate.ConditionList {
				return policyConditionsStatus(TestPolicy)
			}, 2*time.Second, 100*time.Millisecond).Should(containPolicyDegraded(), "policy should be marked as Degraded")
		})
	})
})
