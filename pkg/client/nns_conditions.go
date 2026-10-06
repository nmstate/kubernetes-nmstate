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

package client

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

// SetNodeNetworkStateQueryConditions sets the Available and Failing
// conditions according to the result of the last network state query.
// Reason NodeNetworkStateConditionQuerySucceeded marks the state as
// available, any other reason marks it as failing.
//
// It returns false and leaves the conditions untouched (including
// LastHeartbeatTime) if nothing would change, so callers can skip writing to
// the API server.
func SetNodeNetworkStateQueryConditions(conditions *shared.ConditionList, reason shared.ConditionReason, message string) bool {
	availableStatus, failingStatus := corev1.ConditionFalse, corev1.ConditionTrue
	if reason == shared.NodeNetworkStateConditionQuerySucceeded {
		availableStatus, failingStatus = corev1.ConditionTrue, corev1.ConditionFalse
	}

	if conditionMatches(*conditions, shared.NodeNetworkStateConditionAvailable, availableStatus, reason, message) &&
		conditionMatches(*conditions, shared.NodeNetworkStateConditionFailing, failingStatus, reason, message) {
		return false
	}

	conditions.Set(shared.NodeNetworkStateConditionAvailable, availableStatus, reason, message)
	conditions.Set(shared.NodeNetworkStateConditionFailing, failingStatus, reason, message)
	return true
}

func conditionMatches(
	conditions shared.ConditionList,
	conditionType shared.ConditionType,
	status corev1.ConditionStatus,
	reason shared.ConditionReason,
	message string,
) bool {
	condition := conditions.Find(conditionType)
	return condition != nil && condition.Status == status && condition.Reason == reason && condition.Message == message
}

// UpdateNodeNetworkStateQueryConditions sets the query conditions on the NNS
// and writes its status only if they changed. The rest of the status,
// including the last successfully retrieved current state, is kept as is.
func UpdateNodeNetworkStateQueryConditions(
	ctx context.Context,
	cli client.Client,
	nns *nmstatev1beta1.NodeNetworkState,
	reason shared.ConditionReason,
	message string,
) error {
	if !SetNodeNetworkStateQueryConditions(&nns.Status.Conditions, reason, message) {
		return nil
	}
	if err := cli.Status().Update(ctx, nns); err != nil {
		return errors.Wrap(err, "error updating NodeNetworkState conditions")
	}
	return nil
}

// QuerySucceededMessage is the condition message used after a successful
// network state query.
const QuerySucceededMessage = "Network state successfully retrieved"
