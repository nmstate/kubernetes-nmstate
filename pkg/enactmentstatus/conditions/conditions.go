/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func New(cli client.Client, enactmentKey types.NamespacedName) EnactmentConditions {
	conditions := EnactmentConditions{
		client:       cli,
		enactmentKey: enactmentKey,
		logger:       logf.Log.WithName("enactmentconditions").WithValues("enactment", enactmentKey.Name),
	}
	return conditions
}

func (ec *EnactmentConditions) NotifyGenerateFailure(err error) {
	ec.logger.Info("NotifyGenerateFailure")
	message := fmt.Sprintf("failure generating desiredState and capturedStates: %v", err)
	err = ec.updateEnactmentConditions(SetFailedToConfigure, message)
	if err != nil {
		ec.logger.Error(err, "Error notifying state generate captures with failure")
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

func (ec *EnactmentConditions) NotifyPending() {
	ec.logger.Info("NotifyPending")
	err := ec.updateEnactmentConditions(SetPending, "Max unavailable node limit reached")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Pending")
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
	strippedMessage := enactmentstatus.FormatErrorString(message)
	SetFailed(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, strippedMessage)
	setFailedToConfigureEncodedMessage(conditions, message)
}

func setFailedToConfigureEncodedMessage(conditions *nmstate.ConditionList, message string) {
	condition := conditions.Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing)
	condition.MessageEncoded = enactmentstatus.CompressAndEncodeMessage(message)
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
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
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
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
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
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
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
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
		corev1.ConditionFalse,
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

func SetPending(conditions *nmstate.ConditionList, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		"",
	)
}
