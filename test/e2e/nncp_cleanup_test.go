package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NNCP cleanup", func() {

	BeforeEach(func() {
		By("Create a policy")
		setDesiredStateWithPolicy(bridge1, linuxBrUp(bridge1))

		By("Wait for policy to be ready")
		policyConditionsStatusForPolicyEventually(bridge1).Should(ContainElement(
			nmstatev1alpha1.Condition{
				Type:   nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
				Status: corev1.ConditionTrue,
			},
		))
	})

	AfterEach(func() {
		updateDesiredState(linuxBrAbsent(bridge1))
		policyConditionsStatusEventually().Should(ContainElement(
			nmstatev1alpha1.Condition{
				Type:   nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
				Status: corev1.ConditionTrue,
			},
		))
		resetDesiredStateForNodes()
	})

	Context("when a policy is deleted", func() {
		BeforeEach(func() {
			deletePolicy(bridge1)
		})
		It("should also delete nodes enactments", func() {
			for _, node := range nodes {
				Eventually(func() bool {
					key := nmstatev1alpha1.EnactmentKey(node, bridge1)
					enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
					err := framework.Global.Client.Get(context.TODO(), key, &enactment)
					return errors.IsNotFound(err)
				}, 10*time.Second, 1*time.Second).Should(BeTrue(), "Enactment has not being deleted")
			}
		})
	})
})
