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
	client "sigs.k8s.io/controller-runtime/pkg/client"
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
		err := policyReader.Get(context.TODO(), policyKey, policy)
		if err != nil {
			return errors.Wrap(err, "getting policy failed")
		}

		enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
		policyLabelFilter := client.MatchingLabels{nmstate.EnactmentPolicyLabel: policy.Name}
		err = apiReader.List(context.TODO(), &enactments, policyLabelFilter)
		if err != nil {
			return errors.Wrap(err, "getting enactments failed")
		}

		// Count only nodes that runs nmstate handler and match the policy
		// nodeSelector, could be that users don't want to run knmstate at control-plane for example
		// so they don't want to change net config there.
		nmstateMatchingNodes, err := node.NodesRunningNmstate(apiReader, policy.Spec.NodeSelector)
		if err != nil {
			return errors.Wrap(err, "getting nodes running kubernets-nmstate pods failed")
		}
		numberOfNmstateMatchingNodes := len(nmstateMatchingNodes)
		numberOfReadyNmstateMatchingNodes := len(node.FilterReady(nmstateMatchingNodes))
		numberOfNotReadyNmstateMatchingNodes := numberOfNmstateMatchingNodes - numberOfReadyNmstateMatchingNodes

		// Let's get conditions with true status count filtered by policy generation
		enactmentsCountByCondition := enactmentconditions.Count(enactments, policy.Generation)

		numberOfFinishedEnactments := enactmentsCountByCondition.Available() +
			enactmentsCountByCondition.Failed() +
			enactmentsCountByCondition.Aborted()

		logger.Info(
			fmt.Sprintf("numberOfNmstateMatchingNodes: %d, enactments count: %s",
				numberOfNmstateMatchingNodes,
				enactmentsCountByCondition),
		)

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

		if numberOfNmstateMatchingNodes == 0 {
			message = "Policy does not match any node"
			SetPolicyNotMatching(&policy.Status.Conditions, message)
		} else if enactmentsCountByCondition.Failed() > 0 || enactmentsCountByCondition.Aborted() > 0 {
			message = fmt.Sprintf(
				"%d/%d nodes failed to configure",
				enactmentsCountByCondition.Failed(),
				numberOfNmstateMatchingNodes,
			)
			informOfAbortedEnactments(enactmentsCountByCondition.Aborted())
			SetPolicyFailedToConfigure(&policy.Status.Conditions, message)
		} else if numberOfFinishedEnactments < numberOfReadyNmstateMatchingNodes {
			message = fmt.Sprintf(
				"Policy is progressing %d/%d nodes finished",
				numberOfFinishedEnactments,
				numberOfReadyNmstateMatchingNodes,
			)
			informOfNotReadyNodes(numberOfNotReadyNmstateMatchingNodes)
			SetPolicyProgressing(
				&policy.Status.Conditions,
				message,
			)
		} else {
			message = fmt.Sprintf(
				"%d/%d nodes successfully configured",
				enactmentsCountByCondition.Available(),
				numberOfNmstateMatchingNodes,
			)
			informOfNotReadyNodes(numberOfNotReadyNmstateMatchingNodes)
			SetPolicySuccess(&policy.Status.Conditions, message)
		}

		err = apiWriter.Status().Update(context.TODO(), policy)
		if err != nil {
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
