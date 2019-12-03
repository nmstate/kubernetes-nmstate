package conditions

import (
	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func setPolicyProgressing(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
}
