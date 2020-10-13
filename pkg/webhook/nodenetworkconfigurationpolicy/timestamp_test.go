package nodenetworkconfigurationpolicy

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

func expectTimestampAnnotationAtPolicy(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, testStartTime time.Time) {
	ExpectWithOffset(1, policy.ObjectMeta.Annotations).To(HaveKey(TimestampLabelKey))
	annotation := policy.ObjectMeta.Annotations[TimestampLabelKey]
	mutationTimestamp, err := strconv.ParseInt(annotation, 10, 64)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "mutation timestamp should have int64 value")
	ExpectWithOffset(1, mutationTimestamp).To(BeNumerically(">", testStartTime.UnixNano()), "mutation timestamp should be updated by the webhook")
}

var _ = Describe("NNCP Conditions Mutating Admission Webhook", func() {
	var (
		testStartTime    time.Time
		obtainedResponse webhook.AdmissionResponse
		policy           = nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	)
	BeforeEach(func() {
		testStartTime = time.Now()
	})
	Context("when setTimestampAnnotationHook is called", func() {
		BeforeEach(func() {
			request := requestForPolicy(policy)
			obtainedResponse = setTimestampAnnotationHook().Handle(context.TODO(), request)
		})
		It("should return json patches", func() {
			for _, patch := range obtainedResponse.Patches {
				_, err := patch.MarshalJSON()
				Expect(err).ToNot(HaveOccurred(), "The patches should contain valid JSON")
			}
		})
		It("should return allowed response", func() {
			Expect(obtainedResponse.Allowed).To(BeTrue(), "Mutation of the request should be allowed")
		})
		It("should mark the policy with a timestamp", func() {
			patchedPolicy := patchPolicy(policy, obtainedResponse)
			expectTimestampAnnotationAtPolicy(patchedPolicy, testStartTime)
		})
	})
})
