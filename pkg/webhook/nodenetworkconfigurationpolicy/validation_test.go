package nodenetworkconfigurationpolicy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/policyconditions"
)

func p(nodeSelector map[string]string, conditionsSetter func(*shared.ConditionList, string), message string) nmstatev1.NodeNetworkConfigurationPolicy {
	conditions := shared.ConditionList{}
	conditionsSetter(&conditions, message)
	return nmstatev1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testPolicy",
		},
		Spec: shared.NodeNetworkConfigurationPolicySpec{
			NodeSelector: nodeSelector,
		},
		Status: shared.NodeNetworkConfigurationPolicyStatus{
			Conditions: conditions,
		},
	}
}

var _ = Describe("NNCP Conditions Validation Admission Webhook", func() {
	var allNodes = map[string]string{}
	var testPolicy = nmstatev1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testPolicy",
		},
		Spec:   shared.NodeNetworkConfigurationPolicySpec{},
		Status: shared.NodeNetworkConfigurationPolicyStatus{},
	}
	type ValidationWebhookCase struct {
		policy           nmstatev1.NodeNetworkConfigurationPolicy
		currentPolicy    nmstatev1.NodeNetworkConfigurationPolicy
		validationFn     func(policy nmstatev1.NodeNetworkConfigurationPolicy, current nmstatev1.NodeNetworkConfigurationPolicy) []metav1.StatusCause
		validationResult []metav1.StatusCause
	}
	DescribeTable("the NNCP conditions", func(v ValidationWebhookCase) {
		validationResult := v.validationFn(v.policy, v.currentPolicy)
		Expect(validationResult).To(Equal(v.validationResult))
	},
		Entry("current policy in progress", ValidationWebhookCase{
			policy:        testPolicy,
			currentPolicy: p(allNodes, policyconditions.SetPolicyProgressing, ""),
			validationFn:  validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{
				{
					Message: "policy testPolicy is still in progress",
				},
			},
		}),
		Entry("current policy successfully configured", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(allNodes, policyconditions.SetPolicySuccess, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("current policy not matching", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(allNodes, policyconditions.SetPolicyNotMatching, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("current policy failed to configure", ValidationWebhookCase{
			policy:           testPolicy,
			currentPolicy:    p(allNodes, policyconditions.SetPolicyFailedToConfigure, ""),
			validationFn:     validatePolicyNotInProgressHook,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("policy has invalid node selector key", ValidationWebhookCase{
			policy:       p(map[string]string{"bad key": "bar"}, policyconditions.SetPolicySuccess, ""),
			validationFn: validatePolicyNodeSelector,
			validationResult: []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid label key: \"bad key\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
				Field:   "spec.nodeSelector",
			}},
		}),
		Entry("policy has node selector value with length beyond the limit", ValidationWebhookCase{
			policy:       p(map[string]string{"kubernetes.io/hostname": "this-is-longer-than-sixty-three-characters-hostname-bar-bar-bar.foo.com"}, policyconditions.SetPolicySuccess, ""),
			validationFn: validatePolicyNodeSelector,
			validationResult: []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid label value: \"this-is-longer-than-sixty-three-characters-hostname-bar-bar-bar.foo.com\": at key: \"kubernetes.io/hostname\": must be no more than 63 characters",
				Field:   "spec.nodeSelector",
			}},
		}),
		Entry("policy has node selector value with invalid format", ValidationWebhookCase{
			policy:       p(map[string]string{"kubernetes.io/hostname": "foo+bar"}, policyconditions.SetPolicySuccess, ""),
			validationFn: validatePolicyNodeSelector,
			validationResult: []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid label value: \"foo+bar\": at key: \"kubernetes.io/hostname\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
				Field:   "spec.nodeSelector",
			}},
		}),
		Entry("policy has valid node selector", ValidationWebhookCase{
			policy:           p(map[string]string{"kubernetes.io/hostname": "node01"}, policyconditions.SetPolicySuccess, ""),
			validationFn:     validatePolicyNodeSelector,
			validationResult: []metav1.StatusCause{},
		}),
		Entry("policy has name with length beyond the limit", ValidationWebhookCase{
			policy:       nmstatev1.NodeNetworkConfigurationPolicy{ObjectMeta: metav1.ObjectMeta{Name: "this-is-longer-than-sixty-three-characters-hostname-bar-bar-bar.foo.com"}},
			validationFn: validatePolicyName,
			validationResult: []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid policy name: \"this-is-longer-than-sixty-three-characters-hostname-bar-bar-bar.foo.com\": must be no more than 63 characters",
				Field:   "name",
			}},
		}),
		Entry("policy has name with invalid format", ValidationWebhookCase{
			policy:       nmstatev1.NodeNetworkConfigurationPolicy{ObjectMeta: metav1.ObjectMeta{Name: "foo+bar"}},
			validationFn: validatePolicyName,
			validationResult: []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid policy name: \"foo+bar\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
				Field:   "name",
			}},
		}),
	)
})
