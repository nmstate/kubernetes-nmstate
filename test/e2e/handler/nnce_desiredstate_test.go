package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("Enactment DesiredState", func() {
	Context("when applying a policy to matching nodes", func() {
		BeforeEach(func() {
			By("Create a policy")
			updateDesiredState(linuxBrUp(bridge1))
			policyConditionsStatusEventually().Should(ContainElement(
				nmstatev1alpha1.Condition{
					Type:   nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
					Status: corev1.ConditionTrue,
				},
			))
		})
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredState(linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should have desiredState for node", func() {
			for _, node := range nodes {
				enactmentKey := nmstatev1alpha1.EnactmentKey(node, TestPolicy)
				By(fmt.Sprintf("Check enactment %s has expected desired state", enactmentKey.Name))
				nnce := nodeNetworkConfigurationEnactment(enactmentKey)
				Expect(nnce.Status.DesiredState).To(MatchYAML(linuxBrUp(bridge1)))
			}
		})
	})
})
