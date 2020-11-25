package handler

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("NNCP with parallel set to true", func() {
	Context("when applying a policy to matching nodes", func() {
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
		BeforeEach(func() {
			node := nodes[0]
			By("Create a policy")
			updateDesiredState(linuxBrUp(bridge1))
			By(fmt.Sprintf("Wait for %s progressing state reached", node))
			enactmentConditionsStatusEventually(node).Should(ConsistOf(progressConditions))
		})
		AfterEach(func() {
			By("Remove the policy")
			deletePolicy(TestPolicy)
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should be progressing on multiple nodes at the same time", func() {
			if !parallelRollout {
				Skip("Parallel rollout need to be enabled")
			}
			progressingEnactments := 0

			var wg sync.WaitGroup
			wg.Add(len(nodes))
			for i, _ := range nodes {
				node := nodes[i]
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					enactmentConditionsStatusEventually(node).Should(ConsistOf(progressConditions))
				}()
			}
			wg.Wait()

			for _, node := range nodes {
				enactment := enactmentConditionsStatus(node, TestPolicy)
				if enactment.Find(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing) != nil {
					progressingEnactments++
				}
			}
			By("Check that all node enactments turned progressing before others turned available")
			Expect(progressingEnactments).Should(BeNumerically(">", 1))
		})
	})
})
