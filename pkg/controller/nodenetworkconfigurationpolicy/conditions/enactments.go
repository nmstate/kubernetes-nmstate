package conditions

import (
	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func setEnactmentFailedToConfigure(enactment *nmstatev1alpha1.NodeNetworkConfigurationEnactment, message string) {
	setEnactmentFailed(enactment, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func setEnactmentFailed(enactment *nmstatev1alpha1.NodeNetworkConfigurationEnactment, reason nmstatev1alpha1.ConditionReason, message string) {
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
	enactment.Status.Phase = nmstatev1alpha1.NodeNetworkConfigurationEnactmentPhaseFailing
}

func setEnactmentSuccess(enactment *nmstatev1alpha1.NodeNetworkConfigurationEnactment, message string) {
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		message,
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	enactment.Status.Phase = nmstatev1alpha1.NodeNetworkConfigurationEnactmentPhaseAvailable
}

func setEnactmentProgressing(enactment *nmstatev1alpha1.NodeNetworkConfigurationEnactment, message string) {
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	enactment.Status.Conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	enactment.Status.Phase = nmstatev1alpha1.NodeNetworkConfigurationEnactmentPhaseProgressing
}
