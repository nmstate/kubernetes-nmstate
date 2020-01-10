package mutating

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("Mutating Admission Webhook", func() {
	type WebhookCase struct {
		Policy nmstatev1alpha1.NodeNetworkConfigurationPolicy
	}
	DescribeTable("when reset condition is called", func(c WebhookCase) {
		var (
			testStartTime    int64
			request          = webhook.AdmissionRequest{}
			obtainedResponse webhook.AdmissionResponse
			patchedPolicy    = nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
		)
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion,
			&nmstatev1alpha1.NodeNetworkConfigurationPolicy{},
		)
		c.Policy.Status.Conditions.Set(
			nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
			corev1.ConditionTrue,
			nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
			"")

		cli := fake.NewFakeClientWithScheme(s, &c.Policy)
		testStartTime = time.Now().UnixNano()

		By("invoking the webhook")
		data, err := json.Marshal(c.Policy)
		Expect(err).ToNot(HaveOccurred())
		request.Object = runtime.RawExtension{
			Raw: data,
		}
		obtainedResponse = resetConditionsHook().Handle(context.TODO(), request)

		By("patching the policy with the result")
		patch := client.ConstantPatch(types.JSONPatchType, obtainedResponse.Patch)
		policy := c.Policy.DeepCopy()
		err = cli.Patch(context.TODO(), policy, patch)
		Expect(err).ToNot(HaveOccurred())

		By("Retrieve the patched policy")
		err = cli.Get(context.TODO(), types.NamespacedName{Name: ""}, &patchedPolicy)
		Expect(err).ToNot(HaveOccurred())

		Expect(obtainedResponse.Allowed).To(BeTrue(), "Mutation not allowed")
		Expect(obtainedResponse.Result.Reason).To(Equal(metav1.StatusReason("Conditions reset")))

		Expect(obtainedResponse.Patches).To(HaveLen(2), "Unexpected number of patches")

		for _, patch := range obtainedResponse.Patches {
			_, err := patch.MarshalJSON()
			Expect(err).ToNot(HaveOccurred(), "Invalid patch")
		}

		Expect(patchedPolicy.Status.Conditions).To(BeEmpty(), "Condition list not resetted")

		if c.Policy.ObjectMeta.Annotations != nil {
			Expect(patchedPolicy.ObjectMeta.Annotations).To(HaveLen(len(c.Policy.ObjectMeta.Annotations) + 1))
		}

		Expect(patchedPolicy.ObjectMeta.Annotations).To(HaveKey(TimestampLabelKey))
		annotation := patchedPolicy.ObjectMeta.Annotations[TimestampLabelKey]
		mutationTimestamp, err := strconv.ParseInt(annotation, 10, 64)
		Expect(err).ToNot(HaveOccurred(), "mutation timestamp has not a int64 value")
		Expect(mutationTimestamp).To(BeNumerically(">", testStartTime), "mutation timestamp not updated")

	},
		Entry("when conditions and annotations are empty should add mutation annotation", WebhookCase{
			Policy: nmstatev1alpha1.NodeNetworkConfigurationPolicy{},
		}),
		Entry("when conditions are not empty it should reset them", WebhookCase{
			Policy: nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				Status: nmstatev1alpha1.NodeNetworkConfigurationPolicyStatus{
					Conditions: nmstatev1alpha1.ConditionList{
						nmstatev1alpha1.Condition{},
					},
				},
			},
		}),
		Entry("when annotations are nil it should create them and add expected annotation", WebhookCase{
			Policy: nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
		}),
		Entry("when annotations are empty it should create them and add expected annotation", WebhookCase{
			Policy: nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		}),
		Entry("when annotations are not empty it should add expected annotation", WebhookCase{
			Policy: nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
			},
		}),
	)

})
