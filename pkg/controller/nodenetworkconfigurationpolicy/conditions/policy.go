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

func setPolicySuccess(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		message,
	)
}

func setPolicyNotMatching(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationNotMatchingNode,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationNotMatchingNode,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionConfigurationNotMatchingNode,
		message,
	)
}

func setPolicyFailedToConfigure(conditions *nmstatev1alpha1.ConditionList, message string) {
	setPolicyFailed(conditions, nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailedToConfigure, message)
}

func setPolicyFailed(conditions *nmstatev1alpha1.ConditionList, reason nmstatev1alpha1.ConditionReason, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
}
