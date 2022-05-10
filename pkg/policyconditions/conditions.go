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

package policyconditions

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
)

var (
	log = logf.Log.WithName("policyconditions")
)

type policyConditionStatus struct {
	numberOfNmstateMatchingNodes         int
	numberOfReadyNmstateMatchingNodes    int
	numberOfNotReadyNmstateMatchingNodes int
	enactmentsCountByCondition           enactmentconditions.ConditionCount
	numberOfFinishedEnactments           int
}

func SetPolicyProgressing(conditions *nmstate.ConditionList, message string) {
	log.Info("SetPolicyProgressing")
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionUnknown,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		message,
	)
}

func SetPolicySuccess(conditions *nmstate.ConditionList, message string) {
	log.Info("SetPolicySuccess")
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
}

func SetPolicyNotMatching(conditions *nmstate.ConditionList, message string) {
	log.Info("SetPolicyNotMatching")
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationNoMatchingNode,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationNoMatchingNode,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
}

func SetPolicyFailedToConfigure(conditions *nmstate.ConditionList, message string) {
	log.Info("SetPolicyFailedToConfigure")
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionDegraded,
		corev1.ConditionTrue,
		nmstate.NodeNetworkConfigurationPolicyConditionFailedToConfigure,
		message,
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionFailedToConfigure,
		"",
	)
	conditions.Set(
		nmstate.NodeNetworkConfigurationPolicyConditionProgressing,
		corev1.ConditionFalse,
		nmstate.NodeNetworkConfigurationPolicyConditionConfigurationProgressing,
		"",
	)
}

func IsProgressing(conditions *nmstate.ConditionList) bool {
	progressingCondition := conditions.Find(nmstate.NodeNetworkConfigurationPolicyConditionProgressing)
	if progressingCondition == nil {
		return false
	}
	return progressingCondition.Status == corev1.ConditionTrue
}

func IsUnknown(conditions *nmstate.ConditionList) bool {
	availableCondition := conditions.Find(nmstate.NodeNetworkConfigurationPolicyConditionAvailable)
	if availableCondition == nil {
		return true
	}
	return availableCondition.Status == corev1.ConditionUnknown
}

func Update(cli client.Client, apiReader client.Reader, policyKey types.NamespacedName) error {
	logger := log.WithValues("policy", policyKey.Name)

	err := update(cli, apiReader, cli, policyKey)
	if err != nil {
		logger.Error(err, "failed to update policy status using cached client. Retrying with non-cached.")
		err = update(cli, apiReader, apiReader, policyKey)
		if err != nil {
			logger.Error(err, "failed to update policy status using non-cached client.")
		}
	}
	return err
}

//nolint:gocritic
func update(apiWriter client.Client, apiReader client.Reader, policyReader client.Reader, policyKey types.NamespacedName) error {
	logger := log.WithValues("policy", policyKey.Name)
	// On conflict we need to re-retrieve enactments since the
	// conflict can denote that the calculated policy conditions
	// are now not accurate.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
		if err := policyReader.Get(context.TODO(), policyKey, policy); err != nil {
			return errors.Wrap(err, "getting policy failed")
		}

		enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
		policyLabelFilter := client.MatchingLabels{nmstate.EnactmentPolicyLabel: policy.Name}
		if err := apiReader.List(context.TODO(), &enactments, policyLabelFilter); err != nil {
			return errors.Wrap(err, "getting enactments failed")
		}

		// Count only nodes that runs nmstate handler and match the policy
		// nodeSelector, could be that users don't want to run knmstate at control-plane for example
		// so they don't want to change net config there.
		nmstateMatchingNodes, err := node.NodesRunningNmstate(apiReader, policy.Spec.NodeSelector)
		if err != nil {
			return errors.Wrap(err, "getting nodes running kubernets-nmstate pods failed")
		}

		policyStatus := calculatePolicyConditionStatus(policy, &nmstateMatchingNodes, &enactments)
		logger.Info(
			fmt.Sprintf("numberOfNmstateMatchingNodes: %d, enactments count: %s",
				policyStatus.numberOfNmstateMatchingNodes,
				policyStatus.enactmentsCountByCondition),
		)

		setPolicyStatus(policy, &policyStatus)

		if err = apiWriter.Status().Update(context.TODO(), policy); err != nil {
			if apierrors.IsConflict(err) {
				logger.Info("conflict updating policy conditions, retrying")
			} else {
				logger.Error(err, "failed to update policy conditions")
			}
			return err
		}
		return nil
	})
}

func setPolicyStatus(policy *nmstatev1.NodeNetworkConfigurationPolicy, policyStatus *policyConditionStatus) {
	var message string
	informOfNotReadyNodes := func(notReadyNodesCount int) {
		if notReadyNodesCount > 0 {
			message += fmt.Sprintf(
				", %d nodes ignored due to NotReady state",
				notReadyNodesCount,
			)
		}
	}
	informOfAbortedEnactments := func(abortedEnactmentsCount int) {
		if abortedEnactmentsCount > 0 {
			message += fmt.Sprintf(
				", %d nodes aborted configuration",
				abortedEnactmentsCount,
			)
		}
	}

	if policyStatus.numberOfNmstateMatchingNodes == 0 {
		message = "Policy does not match any node"
		SetPolicyNotMatching(&policy.Status.Conditions, message)
	} else if policyStatus.enactmentsCountByCondition.Failed() > 0 || policyStatus.enactmentsCountByCondition.Aborted() > 0 {
		message = fmt.Sprintf(
			"%d/%d nodes failed to configure",
			policyStatus.enactmentsCountByCondition.Failed(),
			policyStatus.numberOfNmstateMatchingNodes,
		)
		informOfAbortedEnactments(policyStatus.enactmentsCountByCondition.Aborted())
		SetPolicyFailedToConfigure(&policy.Status.Conditions, message)
	} else if policyStatus.numberOfFinishedEnactments < policyStatus.numberOfReadyNmstateMatchingNodes {
		message = fmt.Sprintf(
			"Policy is progressing %d/%d nodes finished",
			policyStatus.numberOfFinishedEnactments,
			policyStatus.numberOfReadyNmstateMatchingNodes,
		)
		informOfNotReadyNodes(policyStatus.numberOfNotReadyNmstateMatchingNodes)
		SetPolicyProgressing(
			&policy.Status.Conditions,
			message,
		)
	} else {
		message = fmt.Sprintf(
			"%d/%d nodes successfully configured",
			policyStatus.enactmentsCountByCondition.Available(),
			policyStatus.numberOfNmstateMatchingNodes,
		)
		informOfNotReadyNodes(policyStatus.numberOfNotReadyNmstateMatchingNodes)
		SetPolicySuccess(&policy.Status.Conditions, message)
	}
}

func calculatePolicyConditionStatus(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	nmstateMatchingNodes *[]corev1.Node,
	enactments *nmstatev1beta1.NodeNetworkConfigurationEnactmentList,
) policyConditionStatus {
	numberOfNmstateMatchingNodes := len(*nmstateMatchingNodes)
	numberOfReadyNmstateMatchingNodes := len(node.FilterReady(*nmstateMatchingNodes))
	// Let's get conditions with true status count filtered by policy generation
	enactmentsCountByCondition := enactmentconditions.Count(*enactments, policy.Generation)

	return policyConditionStatus{
		numberOfNmstateMatchingNodes:         numberOfNmstateMatchingNodes,
		numberOfReadyNmstateMatchingNodes:    numberOfReadyNmstateMatchingNodes,
		numberOfNotReadyNmstateMatchingNodes: numberOfNmstateMatchingNodes - numberOfReadyNmstateMatchingNodes,
		enactmentsCountByCondition:           enactmentsCountByCondition,
		numberOfFinishedEnactments: enactmentsCountByCondition.Available() +
			enactmentsCountByCondition.Failed() +
			enactmentsCountByCondition.Aborted()}
}

func Reset(cli client.Client, policyKey types.NamespacedName) error {
	logger := log.WithValues("policy", policyKey.Name)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
		err := cli.Get(context.TODO(), policyKey, policy)
		if err != nil {
			return errors.Wrap(err, "getting policy failed")
		}
		policy.Status.Conditions = nmstate.ConditionList{}
		err = cli.Status().Update(context.TODO(), policy)
		if err != nil {
			if apierrors.IsConflict(err) {
				logger.Info("conflict resetting policy conditions, retrying")
			} else {
				logger.Error(err, "failed to reset policy conditions")
			}
			return err
		}
		return nil
	})
}
