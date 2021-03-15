package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func enactmentsInProgress(policy string) int {
	progressingEnactments := 0
	for _, node := range nodes {
		enactment := enactmentConditionsStatus(node, policy)
		if cond := enactment.Find(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing); cond != nil {
			if cond.Status == corev1.ConditionTrue {
				progressingEnactments++
			}
		}
	}
	return progressingEnactments
}

var _ = Describe("NNCP with maxUnavailable", func() {
	Context("when applying a policy to matching nodes", func() {
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
			Eventually(func() int {
				return enactmentsInProgress(TestPolicy)
			}).Should(BeNumerically("==", maxUnavailable))
			waitForAvailablePolicy(TestPolicy)
		})
		It("[parallel] should never exceed maxUnavailable nodes", func() {
			duration := 10 * time.Second
			interval := 1 * time.Second
			Consistently(func() int {
				return enactmentsInProgress(TestPolicy)
			}, duration, interval).Should(BeNumerically("<=", maxUnavailable))
			waitForAvailablePolicy(TestPolicy)
		})
	})
})
