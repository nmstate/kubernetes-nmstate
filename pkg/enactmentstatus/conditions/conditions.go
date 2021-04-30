package conditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus"
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
	err = ec.updateEnactmentConditions(SetFailedToConfigure, message)
	if err != nil {
		ec.logger.Error(err, "Error notifying state NodeSelectorNotMatching with failure")
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

func (ec *EnactmentConditions) NotifyAborted(failedErr error) {
	ec.logger.Info("NotifyConfigurationAborted")
	err := ec.updateEnactmentConditions(SetConfigurationAborted, failedErr.Error())
	if err != nil {
		ec.logger.Error(err, "Error notifying state ConfigurationAborted")
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
		*conditionList = nil
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
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured,
		"",
	)
}

func SetConfigurationAborted(conditions *nmstate.ConditionList, message string) {
	SetAborted(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationAborted, message)
}

func SetAborted(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
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
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
		corev1.ConditionTrue,
		reason,
		message,
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
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
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
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		"",
	)
}
