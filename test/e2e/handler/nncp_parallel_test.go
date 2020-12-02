package handler

import (
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
			nmstate.Condition{
				Type:   nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
				Status: corev1.ConditionFalse,
			},
		}
		BeforeEach(func() {
			By("Create a policy")
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Remove the policy")
			deletePolicy(TestPolicy)
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("[parallel] should be progressing on multiple nodes", func() {
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
			By("Check that multiple enactments are progressing.")
			Expect(progressingEnactments).Should(BeNumerically(">", 1))
		})
	})
})
