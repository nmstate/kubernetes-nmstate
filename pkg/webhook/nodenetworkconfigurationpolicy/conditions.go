package nodenetworkconfigurationpolicy

import (
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/shared"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var log = logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/conditions")

func deleteConditions(policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) nmstatev1alpha1.NodeNetworkConfigurationPolicy {
	policy.Status.Conditions = nmstate.ConditionList{}
	return policy
}

func setConditionsUnknown(policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) nmstatev1alpha1.NodeNetworkConfigurationPolicy {
	unknownConditions := nmstate.ConditionList{}
	for _, conditionType := range nmstate.NodeNetworkConfigurationPolicyConditionTypes {
		unknownConditions.Set(
			conditionType,
			corev1.ConditionUnknown,
			"", "")
	}
	policy.Status.Conditions = unknownConditions
	return policy
}

func atEmptyConditions(policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) bool {
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
