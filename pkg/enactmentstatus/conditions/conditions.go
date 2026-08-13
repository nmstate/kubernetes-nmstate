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
	"context"
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

func (ec *EnactmentConditions) NotifyGenerateFailure(ctx context.Context, err error) {
	ec.logger.Info("NotifyGenerateFailure")
	message := fmt.Sprintf("failure generating desiredState and capturedStates: %v", err)
	err = ec.updateEnactmentConditions(ctx, SetFailedToConfigure, message)
	if err != nil {
		ec.logger.Error(err, "Error notifying state generate captures with failure")
	}
}

func (ec *EnactmentConditions) NotifyProgressing(ctx context.Context) error {
	ec.logger.Info("NotifyProgressing")
	err := ec.updateEnactmentConditions(ctx, SetProgressing, "Applying desired state")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Progressing")
	}
	return err
}

// NotifyFinalizing records that the desired state has been applied and only the
// post-apply finalization (releasing the maxUnavailable slot and recording
// success) remains. The enactment stays Progressing (a live slot holder) but
// carries the ConfigurationFinalizing reason so a retry can skip the already
// committed apply. See SetFinalizing.
func (ec *EnactmentConditions) NotifyFinalizing(ctx context.Context) error {
	ec.logger.Info("NotifyFinalizing")
	err := ec.updateEnactmentConditions(ctx, SetFinalizing, "Desired state applied, finalizing")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Finalizing")
	}
	return err
}

func (ec *EnactmentConditions) NotifyFailedToConfigure(ctx context.Context, failedErr error) {
	ec.logger.Info("NotifyFailedToConfigure")
	err := ec.updateEnactmentConditions(ctx, SetFailedToConfigure, failedErr.Error())
	if err != nil {
		ec.logger.Error(err, "Error notifying state FailingToConfigure")
	}
}

func (ec *EnactmentConditions) NotifyRetrying(ctx context.Context, failedErr error) {
	ec.logger.Info("NotifyRetrying")
	err := ec.updateEnactmentConditions(ctx, SetRetryAfterFailed, failedErr.Error())
	if err != nil {
		ec.logger.Error(err, "Error notifying state Retrying")
	}
}

func (ec *EnactmentConditions) NotifyAborted(ctx context.Context, failedErr error) {
	ec.logger.Info("NotifyConfigurationAborted")
	err := ec.updateEnactmentConditions(ctx, SetConfigurationAborted, failedErr.Error())
	if err != nil {
		ec.logger.Error(err, "Error notifying state ConfigurationAborted")
	}
}

func (ec *EnactmentConditions) NotifySuccess(ctx context.Context) error {
	ec.logger.Info("NotifySuccess")
	err := ec.updateEnactmentConditions(ctx, SetSuccess, "successfully reconciled")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Success")
	}
	return err
}

func (ec *EnactmentConditions) NotifyPending(ctx context.Context) {
	ec.logger.Info("NotifyPending")
	err := ec.updateEnactmentConditions(ctx, SetPending, "Waiting for progressing nodes to finish")
	if err != nil {
		ec.logger.Error(err, "Error notifying state Pending")
	}
}

func (ec *EnactmentConditions) updateEnactmentConditions(
	ctx context.Context,
	conditionsSetter func(*nmstate.ConditionList, string),
	message string,
) error {
	return enactmentstatus.Update(ctx, ec.client, ec.enactmentKey,
		func(status *nmstate.NodeNetworkConfigurationEnactmentStatus) {
			conditionsSetter(&status.Conditions, message)
		})
}

func SetFailedToConfigure(conditions *nmstate.ConditionList, message string) {
	SetFailed(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionFailedToConfigure, message)
}

func SetRetryAfterFailed(conditions *nmstate.ConditionList, message string) {
	SetRetry(conditions, nmstate.NodeNetworkConfigurationEnactmentConditionRetrying, message)
}

func SetRetry(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
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
	setProgressingWithReason(
		conditions,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationProgressing,
		message,
	)
}

// SetFinalizing keeps the enactment Progressing (so it still counts as a live
// maxUnavailable slot holder) but stamps the ConfigurationFinalizing reason,
// marking that the desired state was already applied and only slot release and
// success recording remain.
func SetFinalizing(conditions *nmstate.ConditionList, message string) {
	setProgressingWithReason(
		conditions,
		nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationFinalizing,
		message,
	)
}

func setProgressingWithReason(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionFailing,
		corev1.ConditionUnknown,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAvailable,
		corev1.ConditionUnknown,
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
		reason,
		"",
	)
}

const interruptedByRestartMessage = "interrupted by handler restart; waiting to be reapplied"

// MarkInterrupted transitions an enactment that was Progressing when the
// handler died to Pending, so slot audits across the cluster no longer see
// it as a live maxUnavailable slot holder, and resets its retry count for
// the given generation so the re-apply is not skipped.
//
// The Pending transition carries the ConfigurationInterrupted reason (not
// MaxUnavailableLimitReached) so consumers can tell a restart-interrupted
// enactment apart from one throttled by the maxUnavailable cap.
func MarkInterrupted(ctx context.Context, cli client.Client, enactmentKey types.NamespacedName, generationKey string) error {
	return enactmentstatus.Update(ctx, cli, enactmentKey,
		func(status *nmstate.NodeNetworkConfigurationEnactmentStatus) {
			SetPendingWithReason(
				&status.Conditions,
				nmstate.NodeNetworkConfigurationEnactmentConditionConfigurationInterrupted,
				interruptedByRestartMessage,
			)
			if status.RetryCount != nil {
				status.RetryCount[generationKey] = 0
			}
		})
}

// SetPending marks the enactment Pending because the maxUnavailable cap
// refused its slot claim.
func SetPending(conditions *nmstate.ConditionList, message string) {
	SetPendingWithReason(
		conditions,
		nmstate.NodeNetworkConfigurationEnactmentConditionMaxUnavailableLimitReached,
		message,
	)
}

// SetPendingWithReason marks the enactment Pending with an explicit reason so
// callers can distinguish why the apply is waiting (e.g. throttled by the
// maxUnavailable cap versus interrupted by a handler restart).
func SetPendingWithReason(conditions *nmstate.ConditionList, reason nmstate.ConditionReason, message string) {
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionPending,
		corev1.ConditionTrue,
		reason,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionAborted,
		corev1.ConditionFalse,
		reason,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationEnactmentConditionProgressing,
		corev1.ConditionFalse,
		reason,
		message,
	)
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
}
