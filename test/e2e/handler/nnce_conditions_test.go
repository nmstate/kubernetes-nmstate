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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	policyconditions "github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
)

var _ = Describe("EnactmentCondition", func() {
	Context("when applying valid config", func() {
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))

			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should go from Progressing to Available", func() {
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					By("Check nnce is progressing")
					policyconditions.EnactmentConditionsStatusEventually(node).WithPolling(500*time.Millisecond).
						Should(policyconditions.MatchConditionsFrom(enactmentconditions.SetProgressing), "should reach progressing state at %s", node)
					By("Check reach success")
					policyconditions.EnactmentConditionsStatusEventually(node).
						Should(policyconditions.MatchConditionsFrom(enactmentconditions.SetSuccess), "should reach available state at %s", node)
					By("Check continue at success")
					policyconditions.EnactmentConditionsStatusConsistently(node).
						Should(policyconditions.MatchConditionsFrom(enactmentconditions.SetSuccess), "should keep available state at %s", node)
				}()
			}
			// Run the policy after we set the nnce conditions assert so we
			// make sure we catch progressing state.
			updateDesiredState(linuxBrUp(bridge1))

			wg.Wait()

			By("Check policy is at available state")
			policyconditions.WaitForAvailableTestPolicy()
		})
	})

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			updateDesiredState(nmstate.NewState(`interfaces:
  - name: bad1
    type: ethernet
    state: up
`))
		})

		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				Byf("Check %s failing state is reached", node)
				policyconditions.EnactmentConditionsStatusEventually(node).Should(
					SatisfyAny(
						policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure),
						policyconditions.MatchConditionsFrom(enactmentconditions.SetConfigurationAborted),
					), "should eventually reach failing or aborted conditions at enactments",
				)
			}
			By("Check policy is at degraded state")
			policyconditions.WaitForDegradedTestPolicy()

			By("Check that the enactment stays in failing state")
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					Byf("Check %s failing state is kept", node)
					policyconditions.EnactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							policyconditions.MatchConditionsFrom(enactmentconditions.SetConfigurationAborted),
						), "should consistently keep failing or aborted conditions at enactments",
					)
				}()
			}
			wg.Wait()
		})

		It("should have up to maxUnavailable Failing and the rest Aborted ConditionType set to true", func() {
			checkEnactmentCounts := func(policy string) {
				failingConditions := 0
				abortedConditions := 0
				for _, node := range nodes {
					conditionList := policyconditions.EnactmentConditionsStatus(node, TestPolicy)
					success, _ := policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure).Match(conditionList)
					if success {
						failingConditions++
					}
					success, _ = policyconditions.MatchConditionsFrom(enactmentconditions.SetConfigurationAborted).Match(conditionList)
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
					policyconditions.EnactmentConditionsStatusEventually(node).Should(
						SatisfyAny(
							policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							policyconditions.MatchConditionsFrom(enactmentconditions.SetConfigurationAborted),
						), "should consistently keep failing or aborted conditions at enactments")
				}()
			}
			wg.Wait()

			By("Check policy is at degraded state")
			policyconditions.WaitForDegradedTestPolicy()

			By("Check that the enactments stay in failing or aborted state")
			wg.Add(len(nodes))
			for i := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					Byf("Check %s failing state is kept", node)
					policyconditions.EnactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure),
							policyconditions.MatchConditionsFrom(enactmentconditions.SetConfigurationAborted),
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
					conditionList := policyconditions.EnactmentConditionsStatus(node, TestPolicy)
					found, _ := policyconditions.MatchConditionsFrom(enactmentconditions.SetFailedToConfigure).Match(conditionList)
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
				return policyconditions.Status(TestPolicy)
			}, 2*time.Second, 100*time.Millisecond).Should(policyconditions.ContainPolicyDegraded(), "policy should be marked as Degraded")
		})
	})
})
