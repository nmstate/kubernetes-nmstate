package handler

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func invalidConfig(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: invalid_state
`, bridgeName))
}

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component]EnactmentCondition", func() {
	var abortedEnactmentConditions = []interface{}{
		shared.Condition{
			Type:   shared.NodeNetworkConfigurationEnactmentConditionFailing,
			Status: corev1.ConditionFalse,
		},
		shared.Condition{
			Type:   shared.NodeNetworkConfigurationEnactmentConditionAvailable,
			Status: corev1.ConditionFalse,
		},
		shared.Condition{
			Type:   shared.NodeNetworkConfigurationEnactmentConditionProgressing,
			Status: corev1.ConditionFalse,
		},
		shared.Condition{
			Type:   shared.NodeNetworkConfigurationEnactmentConditionMatching,
			Status: corev1.ConditionTrue,
		},
		shared.Condition{
			Type:   shared.NodeNetworkConfigurationEnactmentConditionAborted,
			Status: corev1.ConditionTrue,
		},
	}
	Context("when applying valid config", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))

			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("[test_id:3796]should go from Progressing to Available", func() {
			progressConditions := []nmstate.Condition{
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status: corev1.ConditionTrue,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
					Status: corev1.ConditionUnknown,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
					Status: corev1.ConditionUnknown,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
					Status: corev1.ConditionTrue,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
					Status: corev1.ConditionFalse,
				},
			}
			availableConditions := []nmstate.Condition{
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status: corev1.ConditionFalse,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
					Status: corev1.ConditionTrue,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
					Status: corev1.ConditionFalse,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
					Status: corev1.ConditionTrue,
				},
				nmstate.Condition{
					Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
					Status: corev1.ConditionFalse,
				},
			}
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i, _ := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					By(fmt.Sprintf("Check %s progressing state is reached", node))
					enactmentConditionsStatusEventually(node).Should(ConsistOf(progressConditions))

					By(fmt.Sprintf("Check %s available state is the next condition", node))
					enactmentConditionsStatusEventually(node).Should(ConsistOf(availableConditions))

					By(fmt.Sprintf("Check %s available state is kept", node))
					enactmentConditionsStatusConsistently(node).Should(ConsistOf(availableConditions))
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
		var failingEnactmentConditions = []interface{}{
			shared.Condition{
				Type:   shared.NodeNetworkConfigurationEnactmentConditionFailing,
				Status: corev1.ConditionTrue,
			},
			shared.Condition{
				Type:   shared.NodeNetworkConfigurationEnactmentConditionAvailable,
				Status: corev1.ConditionFalse,
			},
			shared.Condition{
				Type:   shared.NodeNetworkConfigurationEnactmentConditionProgressing,
				Status: corev1.ConditionFalse,
			},
			shared.Condition{
				Type:   shared.NodeNetworkConfigurationEnactmentConditionMatching,
				Status: corev1.ConditionTrue,
			},
			shared.Condition{
				Type:   shared.NodeNetworkConfigurationEnactmentConditionAborted,
				Status: corev1.ConditionFalse,
			},
		}

		BeforeEach(func() {
			updateDesiredState(invalidConfig(bridge1))
		})

		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("[test_id:3795][parallel] should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				By(fmt.Sprintf("Check %s failing state is reached", node))
				enactmentConditionsStatusEventually(node).Should(
					SatisfyAny(
						ConsistOf(failingEnactmentConditions...),
						ConsistOf(abortedEnactmentConditions...),
					), "should eventually reach failing or aborted conditions at enactments",
				)
			}
			By("Check policy is at degraded state")
			waitForDegradedTestPolicy()

			By("Check that the enactment stays in failing state")
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i, _ := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					By(fmt.Sprintf("Check %s failing state is kept", node))
					enactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							ConsistOf(failingEnactmentConditions...),
							ConsistOf(abortedEnactmentConditions...),
						), "should consistently keep failing or aborted conditions at enactments",
					)
				}()
			}
			wg.Wait()
		})

		It("[test_id:3795][sequential] should have one Failing the rest Aborted ConditionType set to true", func() {
			checkEnactmentCounts := func(policy string) {
				failingConditions := 0
				abortedConditions := 0
				for _, node := range nodes {
					conditionList := enactmentConditionsStatus(node, TestPolicy)
					success, _ := ConsistOf(conditionList).Match(failingEnactmentConditions)
					if success {
						failingConditions++
					}
					success, _ = ConsistOf(conditionList).Match(abortedEnactmentConditions)
					if success {
						abortedConditions++
					}
				}
				Expect(failingConditions).To(Equal(1), "one node only should have failing enactment")
				Expect(abortedConditions).To(Equal(len(nodes)-1), "other nodes should have aborted enactment")
			}

			By("Check policy is at degraded state")
			waitForDegradedTestPolicy()

			By("Check there is one failing enactment and the rest are aborted")
			checkEnactmentCounts(TestPolicy)

			By("Check that the enactment stays in failing or aborted state")
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i, _ := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					By(fmt.Sprintf("Check %s failing state is kept", node))
					enactmentConditionsStatusConsistently(node).Should(
						SatisfyAny(
							ConsistOf(failingEnactmentConditions...),
							ConsistOf(abortedEnactmentConditions...),
						), "should consistently keep failing or aborted conditions at enactments")
				}()
			}
			wg.Wait()

			By("Check there is still one failing enactment and the rest are aborted")
			checkEnactmentCounts(TestPolicy)
		})
	})
})
