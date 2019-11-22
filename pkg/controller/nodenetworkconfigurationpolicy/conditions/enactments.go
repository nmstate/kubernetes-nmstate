package conditions

import (
	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func setEnactmentFailedToConfigure(policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy, nodeName string, message string) {
	setEnactmentFailed(policy, nodeName, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func setEnactmentFailedToFindPolicy(policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy, nodeName string, message string) {
	setEnactmentFailed(policy, nodeName, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToFindPolicy, message)
}

func setEnactmentFailed(policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy, nodeName string, reason nmstatev1alpha1.ConditionReason, message string) {
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
}

func setEnactmentSuccess(policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy, nodeName string, message string) {
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		message,
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
}

func setEnactmentProgressing(policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy, nodeName string, message string) {
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	policy.SetEnactmentCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
}
