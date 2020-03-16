package e2e

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	runner "github.com/nmstate/kubernetes-nmstate/test/runner"
)

func invalidConfig(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: invalid_state
`, bridgeName))
}

var _ = Describe("EnactmentCondition", func() {
	Context("when applying valid config", func() {
		BeforeEach(func() {
			By("Add some sleep time to vlan-filtering")
			runner.RunAtPods("cp", "/usr/local/bin/vlan-filtering", "/usr/local/bin/vlan-filtering.bak")
			runner.RunAtPods("sed", "-i", "$ a\\sleep 5", "/usr/local/bin/vlan-filtering")
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			By("Restore original vlan-filtering")
			runner.RunAtPods("mv", "/usr/local/bin/vlan-filtering.bak", "/usr/local/bin/vlan-filtering")

			By("Remove the bridge")
			updateDesiredState(linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()

			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		// CNV-3796
		It("should go from Progressing to Available", func() {
			progressConditions := []nmstatev1alpha1.Condition{
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status: corev1.ConditionTrue,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
					Status: corev1.ConditionUnknown,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
					Status: corev1.ConditionUnknown,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching,
					Status: corev1.ConditionTrue,
				},
			}
			availableConditions := []nmstatev1alpha1.Condition{
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status: corev1.ConditionFalse,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
					Status: corev1.ConditionTrue,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
					Status: corev1.ConditionFalse,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching,
					Status: corev1.ConditionTrue,
				},
			}
			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i, _ := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					By(fmt.Sprintf("Check %s progressing state is reached", node))
					enactmentConditionsStatusEventually(node).Should(ConsistOf(progressConditions))

					By(fmt.Sprintf("Check %s available state is the next condition", node))
					enactmentConditionsStatusEventually(node).Should(ConsistOf(availableConditions))

					By(fmt.Sprintf("Check %s available state is kept", node))
					enactmentConditionsStatusConsistently(node).Should(ConsistOf(availableConditions))
				}()
			}
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
			updateDesiredState(linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})

		// CNV-3795
		It("should have Failing ConditionType set to true", func() {
			failingEnactmentConditions := []interface{}{
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
					Status: corev1.ConditionTrue,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
					Status: corev1.ConditionFalse,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status: corev1.ConditionFalse,
				},
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching,
					Status: corev1.ConditionTrue,
				},
			}

			for _, node := range nodes {
				By(fmt.Sprintf("Check %s failing state is reached", node))
				enactmentConditionsStatusEventually(node).Should(ConsistOf(failingEnactmentConditions...))
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
					By(fmt.Sprintf("Check %s failing state is kept", node))
					enactmentConditionsStatusConsistently(node).Should(ConsistOf(failingEnactmentConditions...))
				}()
			}
			wg.Wait()
		})
	})
})
