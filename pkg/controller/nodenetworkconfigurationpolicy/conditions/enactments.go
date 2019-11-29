package conditions

import (
	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func setEnactmentFailedToConfigure(enactments *nmstatev1alpha1.EnactmentList, nodeName string, message string) {
	setEnactmentFailed(enactments, nodeName, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func setEnactmentFailed(enactments *nmstatev1alpha1.EnactmentList, nodeName string, reason nmstatev1alpha1.ConditionReason, message string) {
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
}

func setEnactmentSuccess(enactments *nmstatev1alpha1.EnactmentList, nodeName string, message string) {
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		message,
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
}

func setEnactmentProgressing(enactments *nmstatev1alpha1.EnactmentList, nodeName string, message string) {
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
}
