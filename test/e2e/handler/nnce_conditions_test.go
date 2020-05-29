package handler

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/shared"
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
		BeforeEach(func() {
			updateDesiredState(invalidConfig(bridge1))

		})

		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		It("[test_id:3795]should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				enactmentConditionsStatusEventually(node).Should(ConsistOf(
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
						Status: corev1.ConditionTrue,
					},
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
						Status: corev1.ConditionFalse,
					},
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
						Status: corev1.ConditionTrue,
					},
				))
			}
			By("Check policy is at degraded state")
			waitForDegradedTestPolicy()
		})
	})
})
