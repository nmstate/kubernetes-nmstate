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

func update(apiWriter client.Client, apiReader client.Reader, policyReader client.Reader, policyKey types.NamespacedName) error {
	logger := log.WithValues("policy", policyKey.Name)
	// On conflict we need to re-retrieve enactments since the
	// conflict can denote that the calculated policy conditions
	// are now not accurate.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		policy := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
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

		// Let's get conditions with true status count filtered by policy generation
		enactmentsCountByCondition := enactmentconditions.Count(enactments, policy.Generation)

		numberOfFinishedEnactments := enactmentsCountByCondition.Available() + enactmentsCountByCondition.Failed() + enactmentsCountByCondition.Aborted()

		logger.Info(fmt.Sprintf("numberOfNmstateMatchingNodes: %d, enactments count: %s", numberOfNmstateMatchingNodes, enactmentsCountByCondition))

		if numberOfNmstateMatchingNodes == 0 {
			message := "Policy does not match any node"
			SetPolicyNotMatching(&policy.Status.Conditions, message)
		} else if enactmentsCountByCondition.Failed() > 0 || enactmentsCountByCondition.Aborted() > 0 {
			message := fmt.Sprintf("%d/%d nodes failed to configure", enactmentsCountByCondition.Failed(), numberOfNmstateMatchingNodes)
			if enactmentsCountByCondition.Aborted() > 0 {
				message += fmt.Sprintf(", %d nodes aborted configuration", enactmentsCountByCondition.Aborted())
			}
			SetPolicyFailedToConfigure(&policy.Status.Conditions, message)
		} else if numberOfFinishedEnactments < numberOfNmstateMatchingNodes {
			SetPolicyProgressing(&policy.Status.Conditions, fmt.Sprintf("Policy is progressing %d/%d nodes finished", numberOfFinishedEnactments, numberOfNmstateMatchingNodes))
		} else {
			message := fmt.Sprintf("%d/%d nodes successfully configured", enactmentsCountByCondition.Available(), enactmentsCountByCondition.Available())
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
		policy := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
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
