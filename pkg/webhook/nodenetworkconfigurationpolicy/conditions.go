package nodenetworkconfigurationpolicy

import (
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1beta1"
)

var log = logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/conditions")

func deleteConditions(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	policy.Status.Conditions = nmstatev1beta1.ConditionList{}
	return policy
}

func setConditionsUnknown(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	unknownConditions := nmstatev1beta1.ConditionList{}
	for _, conditionType := range nmstatev1beta1.NodeNetworkConfigurationPolicyConditionTypes {
		unknownConditions.Set(
			conditionType,
			corev1.ConditionUnknown,
			"", "")
	}
	policy.Status.Conditions = unknownConditions
	return policy
}

func atEmptyConditions(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) bool {
	return policy.Status.Conditions == nil || len(policy.Status.Conditions) == 0
}

func deleteConditionsHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				always,
				deleteConditions,
			)),
	}
}

func setConditionsUnknownHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				atEmptyConditions,
				setConditionsUnknown,
			)),
	}
}
