package conditions

import (
	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func setEnactmentFailedToConfigure(conditions *nmstatev1alpha1.ConditionList, message string) {
	setEnactmentFailed(conditions, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func setEnactmentFailed(conditions *nmstatev1alpha1.ConditionList, reason nmstatev1alpha1.ConditionReason, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
}

func setEnactmentSuccess(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
}

func setEnactmentProgressing(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
}
