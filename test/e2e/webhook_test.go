package e2e

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	mutatingwebhook "github.com/nmstate/kubernetes-nmstate/pkg/webhook/mutating"
)

var _ = Describe("Mutating Admission Webhook", func() {
	policyName := "webhook-test"
	Context("when policy is created", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicy(policyName, linuxBrUp(bridge1))
			waitForAvailablePolicy(policyName)
		})
		AfterEach(func() {
			setDesiredStateWithPolicy(policyName, linuxBrAbsent(bridge1))
			waitForAvailablePolicy(policyName)
			deletePolicy(policyName)
		})

		It("should add an annotation with mutation timestamp", func() {
			policy := nodeNetworkConfigurationPolicy(policyName)
			Expect(policy.ObjectMeta.Annotations).To(HaveKey(mutatingwebhook.TimestampLabelKey))
		})
		Context("and we updated it", func() {
			var (
				oldPolicy nmstatev1alpha1.NodeNetworkConfigurationPolicy
			)
			BeforeEach(func() {
				oldPolicy = nodeNetworkConfigurationPolicy(policyName)
				setDesiredStateWithPolicy(policyName, linuxBrAbsent(bridge1))
				waitForAvailablePolicy(policyName)
			})
			It("should update annotation with newer mutation timestamp", func() {
				newPolicy := nodeNetworkConfigurationPolicy(policyName)
				Expect(newPolicy.ObjectMeta.Annotations).To(HaveKey(mutatingwebhook.TimestampLabelKey))

				annotation := oldPolicy.ObjectMeta.Annotations[mutatingwebhook.TimestampLabelKey]
				oldConditionsMutation, err := strconv.ParseInt(annotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				annotation = newPolicy.ObjectMeta.Annotations[mutatingwebhook.TimestampLabelKey]
				newConditionsMutation, err := strconv.ParseInt(annotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())

				Expect(newConditionsMutation).To(BeNumerically(">", oldConditionsMutation), "mutation timestamp not updated")
			})
		})
	})
})
