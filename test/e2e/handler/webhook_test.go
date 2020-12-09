package handler

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nncpwebhook "github.com/nmstate/kubernetes-nmstate/pkg/webhook/nodenetworkconfigurationpolicy"
	"github.com/nmstate/kubernetes-nmstate/test/node"
)

// We just check the labe at CREATE/UPDATE events since mutated data is already
// check at unit test.
var _ = Describe("Mutating Admission Webhook", func() {
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
			Expect(policy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampPolicyLabelKey))
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
				Expect(newPolicy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampPolicyLabelKey))

				oldAnnotation := oldPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampPolicyLabelKey]
				oldConditionsMutation, err := strconv.ParseInt(oldAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				newAnnotation := newPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampPolicyLabelKey]
				newConditionsMutation, err := strconv.ParseInt(newAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())

				Expect(newConditionsMutation).To(BeNumerically(">", oldConditionsMutation), "mutation timestamp not updated")
			})
		})
		Context("and we add a label to a node", func() {
			var (
				testLabel = map[string]string{
					"testKey": "testValue",
				}
			)
			BeforeEach(func() {
				node.AddLabels(nodes[0], testLabel)
			})
			AfterEach(func() {
				node.RemoveLabels(nodes[0], testLabel)
			})
			It("should have two annotations with timestamp", func() {
				policy := nodeNetworkConfigurationPolicy(TestPolicy)
				Expect(policy.ObjectMeta.Annotations).To(SatisfyAll(
					HaveKey(nncpwebhook.TimestampAllPoliciesLabelKey),
					HaveKey(nncpwebhook.TimestampPolicyLabelKey)))
			})
		})
	})
})
