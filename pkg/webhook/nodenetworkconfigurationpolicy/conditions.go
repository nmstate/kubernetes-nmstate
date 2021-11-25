package nodenetworkconfigurationpolicy

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

func deleteConditions(policy nmstatev1.NodeNetworkConfigurationPolicy) nmstatev1.NodeNetworkConfigurationPolicy {
	policy.Status.Conditions = shared.ConditionList{}
	return policy
}

func setConditionsUnknown(policy nmstatev1.NodeNetworkConfigurationPolicy) nmstatev1.NodeNetworkConfigurationPolicy {
	unknownConditions := shared.ConditionList{}
	for _, conditionType := range shared.NodeNetworkConfigurationPolicyConditionTypes {
		unknownConditions.Set(
			conditionType,
			corev1.ConditionUnknown,
			"", "")
	}
	policy.Status.Conditions = unknownConditions
	return policy
}

func atEmptyConditions(policy nmstatev1.NodeNetworkConfigurationPolicy) bool {
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
