package conditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	"github.com/nmstate/kubernetes-nmstate/pkg/controller/nodenetworkconfigurationpolicy/enactmentstatus"
)

type EnactmentConditions struct {
	client       client.Client
	enactmentKey types.NamespacedName
	logger       logr.Logger
}

func New(client client.Client, enactmentKey types.NamespacedName) EnactmentConditions {
	conditions := EnactmentConditions{
		client:       client,
		enactmentKey: enactmentKey,
		logger:       logf.Log.WithName("enactmentconditions").WithValues("enactment", enactmentKey.Name),
	}
	return conditions
}

func (ec *EnactmentConditions) NotifyNodeSelectorFailure(err error) {
	ec.logger.Info("NotifyNodeSelectorFailure")
	message := fmt.Sprintf("failure checking node selectors : %v", err)
	err = ec.updateEnactmentConditions(SetNodeSelectorNotMatching, message)
	if err != nil {
		ec.logger.Error(err, "Error notifying state NodeSelectorNotMatching with failure")
	}
}

func (ec *EnactmentConditions) NotifyNodeSelectorNotMatching(unmatchingLabels map[string]string) {
	ec.logger.Info("NotifyNodeSelectorNotMatching")
	message := fmt.Sprintf("Unmatching labels: %v", unmatchingLabels)
	err := ec.updateEnactmentConditions(SetNodeSelectorNotMatching, message)
	if err != nil {
		ec.logger.Error(err, "Error notifying state NodeSelectorNotMatching")
	}
}

func (ec *EnactmentConditions) NotifyMatching() {
	ec.logger.Info("NotifyMatching")
	err := ec.updateEnactmentConditions(SetMatching, "All policy selectors are matching the node")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Matching")
	}
}

func (ec *EnactmentConditions) NotifyProgressing() {
	ec.logger.Info("NotifyProgressing")
	err := ec.updateEnactmentConditions(SetProgressing, "Applying desired state")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Progressing")
	}
}

func (ec *EnactmentConditions) NotifyFailedToConfigure(failedErr error) {
	ec.logger.Info("NotifyFailedToConfigure")
	err := ec.updateEnactmentConditions(SetFailedToConfigure, failedErr.Error())
	if err != nil {
		ec.logger.Error(err, "Error notifying state FailingToConfigure")
	}
}

func (ec *EnactmentConditions) NotifySuccess() {
	ec.logger.Info("NotifySuccess")
	err := ec.updateEnactmentConditions(SetSuccess, "successfully reconciled")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Success")
	}
}

func (ec *EnactmentConditions) Reset() {
	ec.logger.Info("Reset")
	err := ec.updateEnactmentConditions(func(conditionList *nmstatev1alpha1.ConditionList, message string) {
		conditionList = &nmstatev1alpha1.ConditionList{}
	}, "")
	if err != nil {
		ec.logger.Error(err, "Error resetting conditions")
	}
}

func (ec *EnactmentConditions) updateEnactmentConditions(
	conditionsSetter func(*nmstatev1alpha1.ConditionList, string),
	message string,
) error {
	return enactmentstatus.Update(ec.client, ec.enactmentKey,
		func(status *nmstatev1alpha1.NodeNetworkConfigurationEnactmentStatus) {
			conditionsSetter(&status.Conditions, message)
		})
}

func SetFailedToConfigure(conditions *nmstatev1alpha1.ConditionList, message string) {
	SetFailed(conditions, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func SetFailed(conditions *nmstatev1alpha1.ConditionList, reason nmstatev1alpha1.ConditionReason, message string) {
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

func SetSuccess(conditions *nmstatev1alpha1.ConditionList, message string) {
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

func SetProgressing(conditions *nmstatev1alpha1.ConditionList, message string) {
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

func SetNodeSelectorNotMatching(conditions *nmstatev1alpha1.ConditionList, message string) {
	SetNotMatching(conditions, nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionNodeSelectorNotMatching, message)
}

func SetNotMatching(conditions *nmstatev1alpha1.ConditionList, reason nmstatev1alpha1.ConditionReason, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		reason,
		"",
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
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching,
		corev1.ConditionFalse,
		reason,
		message,
	)
}

func SetMatching(conditions *nmstatev1alpha1.ConditionList, message string) {
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionUnknown,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		message,
	)
}
