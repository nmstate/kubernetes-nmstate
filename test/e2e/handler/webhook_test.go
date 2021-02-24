package handler

import (
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/util/retry"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

// We just check the labe at CREATE/UPDATE events since mutated data is already
// check at unit test.
var _ = Describe("Mutating Admission Webhook", func() {
	var (
		timestampLabelKey = "nmstate.io/webhook-mutating-timestamp"
	)
	Context("when policy is created", func() {
		BeforeEach(func() {
			// Make sure test policy is not there so
			// we exercise CREATE event
			resetDesiredStateForNodes()
			updateDesiredStateAndWait(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})

		It("should have an annotation with mutation timestamp", func() {
			policy := nodeNetworkConfigurationPolicy(TestPolicy)
			Expect(policy.ObjectMeta.Annotations).To(HaveKey(timestampLabelKey))
		})
		Context("and we updated it", func() {
			var (
				oldPolicy nmstatev1beta1.NodeNetworkConfigurationPolicy
			)
			BeforeEach(func() {
				oldPolicy = nodeNetworkConfigurationPolicy(TestPolicy)
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			})
			It("should have an annotation with newer mutation timestamp", func() {
				newPolicy := nodeNetworkConfigurationPolicy(TestPolicy)
				Expect(newPolicy.ObjectMeta.Annotations).To(HaveKey(timestampLabelKey))

				oldAnnotation := oldPolicy.ObjectMeta.Annotations[timestampLabelKey]
				oldConditionsMutation, err := strconv.ParseInt(oldAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				newAnnotation := newPolicy.ObjectMeta.Annotations[timestampLabelKey]
				newConditionsMutation, err := strconv.ParseInt(newAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())

				Expect(newConditionsMutation).To(BeNumerically(">", oldConditionsMutation), "mutation timestamp not updated")
			})
		})
	})
})

var _ = Describe("Validation Admission Webhook", func() {
	Context("When a policy is created and progressing", func() {
		BeforeEach(func() {
			By("Creating a policy without waiting for it to be available")
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			waitForAvailablePolicy(TestPolicy)
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})
		It("Should deny updating sequentially rolled out policy when it's in progress", func() {
			By(fmt.Sprintf("Updating the policy %s", TestPolicy))
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				return setDesiredStateWithPolicyAndNodeSelector(TestPolicy, linuxBrUpNoPorts(bridge1), map[string]string{})
			})
			Expect(err).To(MatchError("admission webhook \"validate-nmstate-io-v1beta-nodenetworkconfigurationpolicy.nmstate.io\" denied the request: policy test-policy is still in progress"))
		})
	})
})
