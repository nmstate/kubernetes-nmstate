package e2e

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	nncpwebhook "github.com/nmstate/kubernetes-nmstate/pkg/webhook/nodenetworkconfigurationpolicy"
)

func expectConditionsUnknown(policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) {
	numberOfConditionTypes := len(nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionTypes)
	ExpectWithOffset(1, policy.Status.Conditions).To(HaveLen(numberOfConditionTypes))
	for _, conditionType := range nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionTypes {
		condition := policy.Status.Conditions.Find(conditionType)
		ExpectWithOffset(1, condition).ToNot(BeNil())
		ExpectWithOffset(1, condition.Status).To(Equal(corev1.ConditionUnknown))
		ExpectWithOffset(1, condition.Reason).To(Equal(nmstatev1alpha1.ConditionReason("")))
		ExpectWithOffset(1, condition.Message).To(Equal(""))
		ExpectWithOffset(1, condition.LastTransitionTime.Time).To(BeTemporally(">", time.Unix(0, 0)))
		ExpectWithOffset(1, condition.LastHeartbeatTime.Time).To(BeTemporally(">", time.Unix(0, 0)))
	}
}

// We just check the labe at CREATE/UPDATE events since mutated data is already
// check at unit test.
var _ = PDescribe("Mutating Admission Webhook [pending openshift not working]", func() {
	Context("when policy is created", func() {
		BeforeEach(func() {
			// Make sure test policy is not there so
			// we exercise CREATE event
			resetDesiredStateForNodes()
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			waitForAvailableTestPolicy()
			updateDesiredState(linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})

		It("should have unknown state and an annotation with mutation timestamp", func() {
			policy := nodeNetworkConfigurationPolicy(TestPolicy)
			expectConditionsUnknown(policy)
			Expect(policy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampLabelKey))
		})
		Context("and we updated it", func() {
			var (
				oldPolicy nmstatev1alpha1.NodeNetworkConfigurationPolicy
			)
			BeforeEach(func() {
				oldPolicy = nodeNetworkConfigurationPolicy(TestPolicy)
				updateDesiredState(linuxBrAbsent(bridge1))
			})
			It("should have unknown state and update annotation with newer mutation timestamp", func() {
				newPolicy := nodeNetworkConfigurationPolicy(TestPolicy)
				expectConditionsUnknown(newPolicy)
				Expect(newPolicy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampLabelKey))

				oldAnnotation := oldPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampLabelKey]
				oldConditionsMutation, err := strconv.ParseInt(oldAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				newAnnotation := newPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampLabelKey]
				newConditionsMutation, err := strconv.ParseInt(newAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())

				Expect(newConditionsMutation).To(BeNumerically(">", oldConditionsMutation), "mutation timestamp not updated")
			})
		})
	})
})
