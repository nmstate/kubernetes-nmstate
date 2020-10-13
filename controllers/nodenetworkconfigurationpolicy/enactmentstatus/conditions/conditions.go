package conditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/enactmentstatus"
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
	err := ec.updateEnactmentConditions(func(conditionList *nmstate.ConditionList, message string) {
		conditionList = &nmstate.ConditionList{}
	}, "")
	if err != nil {
		ec.logger.Error(err, "Error resetting conditions")
	}
}

func (ec *EnactmentConditions) updateEnactmentConditions(
	conditionsSetter func(*nmstate.ConditionList, string),
	message string,
) error {
	return enactmentstatus.Update(ec.client, ec.enactmentKey,
		func(status *nmstate.NodeNetworkConfigurationEnactmentStatus) {
			conditionsSetter(&status.Conditions, message)
		})
}

func SetFailedToConfigure(conditions *nmstate.ConditionList, message string) {
	SetFailed(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func SetFailed(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
}

func SetSuccess(conditions *nmstate.ConditionList, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
}

func SetProgressing(conditions *nmstate.ConditionList, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
}

func SetNodeSelectorNotMatching(conditions *nmstate.ConditionList, message string) {
	SetNotMatching(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionNodeSelectorNotMatching, message)
}

func SetNotMatching(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
		corev1.ConditionFalse,
		reason,
		message,
	)
}

func SetMatching(conditions *nmstate.ConditionList, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching,
		message,
	)
}
