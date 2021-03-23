package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
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

func maxUnavailableNodes() int {
	m, _ := node.ScaledMaxUnavailableNodeCount(len(nodes), intstr.FromString(node.DEFAULT_MAXUNAVAILABLE))
	return m
}

var _ = Describe("NNCP with maxUnavailable", func() {
	policy := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	policy.Name = TestPolicy
	Context("when applying a policy to matching nodes", func() {
		duration := 10 * time.Second
		interval := 1 * time.Second
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
		It("should be progressing on multiple nodes", func() {
			Eventually(func() int {
				return enactmentsInProgress(TestPolicy)
			}, duration, interval).Should(BeNumerically("==", maxUnavailableNodes()))
			waitForAvailablePolicy(TestPolicy)
		})
		It("should never exceed maxUnavailable nodes", func() {
			Consistently(func() int {
				return enactmentsInProgress(TestPolicy)
			}, duration, interval).Should(BeNumerically("<=", maxUnavailableNodes()))
			waitForAvailablePolicy(TestPolicy)
		})
	})
})
