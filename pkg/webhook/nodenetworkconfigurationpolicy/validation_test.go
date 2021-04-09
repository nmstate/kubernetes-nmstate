package nodenetworkconfigurationpolicy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/policyconditions"
)

func p(conditionsSetter func(*shared.ConditionList, string), message string) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	conditions := shared.ConditionList{}
	conditionsSetter(&conditions, message)
	return nmstatev1beta1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testPolicy",
		},
		Status: shared.NodeNetworkConfigurationPolicyStatus{
			Conditions: conditions,
		},
	}
}

var _ = Describe("NNCP Conditions Validation Admission Webhook", func() {
	var testPolicy = nmstatev1beta1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testPolicy",
		},
		Spec:   shared.NodeNetworkConfigurationPolicySpec{},
		Status: shared.NodeNetworkConfigurationPolicyStatus{},
	}
	type ValidationWebhookCase struct {
		policy           nmstatev1beta1.NodeNetworkConfigurationPolicy
		currentPolicy    nmstatev1beta1.NodeNetworkConfigurationPolicy
		validationFn     func(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, current nmstatev1beta1.NodeNetworkConfigurationPolicy) []metav1.StatusCause
		validationResult []metav1.StatusCause
	}
	DescribeTable("the NNCP conditions", func(v ValidationWebhookCase) {
		validationResult := v.validationFn(v.policy, v.currentPolicy)
		Expect(validationResult).To(Equal(v.validationResult))
	},
		Entry("current policy in progress", ValidationWebhookCase{
			policy:        testPolicy,
			currentPolicy: p(policyconditions.SetPolicyProgressing, ""),
			validationFn:  validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{
				{
					Message: "policy testPolicy is still in progress",
				},
			},
		}),
		Entry("current policy successfully configured", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(policyconditions.SetPolicySuccess, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("current policy not matching", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(policyconditions.SetPolicyNotMatching, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("current policy failed to configure", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(policyconditions.SetPolicyFailedToConfigure, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
	)
})
