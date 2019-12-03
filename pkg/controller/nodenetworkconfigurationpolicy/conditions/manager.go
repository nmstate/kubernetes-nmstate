package conditions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Manager struct {
	client   client.Client
	policy   *nmstatev1alpha1.NodeNetworkConfigurationPolicy
	nodeName string
	logger   logr.Logger
}

func NewManager(client client.Client, nodeName string, policy *nmstatev1alpha1.NodeNetworkConfigurationPolicy) Manager {
	manager := Manager{
		client:   client,
		policy:   policy,
		nodeName: nodeName,
	}
	manager.logger = logf.Log.WithName("policy/conditions/manager").WithValues("enactment", nmstatev1alpha1.EnactmentKey(nodeName, policy.Name).Name)
	return manager
}
func (m *Manager) NotifyNodeSelectorFailure(err error) {
	message := fmt.Sprintf("failure checking node selectors for %s: %v", m.nodeName, err)
	err = m.updateEnactmentConditions(setEnactmentNodeSelectorNotMatching, message)
	if err != nil {
		m.logger.Error(err, "Error notifying state NodeSelectorNotMatching with failure")
	}
	m.refreshPolicyConditions()
}
func (m *Manager) NotifyNodeSelectorNotMatching(unmatchingLabels map[string]string) {
	message := fmt.Sprintf("Unmatching labels: %v", unmatchingLabels)
	err := m.updateEnactmentConditions(setEnactmentNodeSelectorNotMatching, message)
	if err != nil {
		m.logger.Error(err, "Error notifying state NodeSelectorNotMatching")
	}
	m.refreshPolicyConditions()
}
func (m *Manager) NotifyMatching() {
	err := m.updateEnactmentConditions(setEnactmentMatching, "All policy selectors are matching the node")
	if err != nil {
		m.logger.Error(err, "Error notifying state Matching")
	}
	m.refreshPolicyConditions()
}
func (m *Manager) NotifyProgressing() {
	err := m.updateEnactmentConditions(setEnactmentProgressing, "Applying desired state")
	if err != nil {
		m.logger.Error(err, "Error notifying state Progressing")
	}
	m.refreshPolicyConditions()
}
func (m *Manager) NotifyFailedToConfigure(failedErr error) {
	err := m.updateEnactmentConditions(setEnactmentFailedToConfigure, failedErr.Error())
	if err != nil {
		m.logger.Error(err, "Error notifying state FailingToConfigure")
	}
	m.refreshPolicyConditions()
}

func (m *Manager) NotifySuccess() {
	err := m.updateEnactmentConditions(setEnactmentSuccess, "successfully reconciled")
	if err != nil {
		m.logger.Error(err, "Error notifying state Success")
	}
	m.refreshPolicyConditions()
}

func (m *Manager) updateEnactmentConditions(
	conditionsSetter func(*nmstatev1alpha1.ConditionList, string),
	message string,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
		err := m.client.Get(context.TODO(), nmstatev1alpha1.EnactmentKey(m.nodeName, m.policy.Name), instance)
		if err != nil {
			return errors.Wrap(err, "getting enactment failed")
		}

		conditionsSetter(&instance.Status.Conditions, message)

		err = m.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return err
		}
		return nil
	})
}

func (m *Manager) refreshPolicyConditions() error {

	// On conflict we need to re-retrieve enactments since the
	// conflict can denote that the calculated policy conditions
	// are now not accurate.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {

		enactments := nmstatev1alpha1.NodeNetworkConfigurationEnactmentList{}
		policyLabelFilter := client.MatchingLabels{"policy": m.policy.Name}
		err := m.client.List(context.TODO(), &enactments, policyLabelFilter)
		if err != nil {
			return errors.Wrap(err, "geting enactments failed")
		}

		nodes := corev1.NodeList{}
		nodeSelectorFilter := client.MatchingLabels(m.policy.Spec.NodeSelector)
		err = m.client.List(context.TODO(), &nodes, nodeSelectorFilter)
		if err != nil {
			return errors.Wrap(err, "geting nodes failed")
		}
		numberOfNodes := len(nodes.Items)
		numberOfEnactments := len(enactments.Items)

		// Let's get conditions with true status frequency
		trueConditionsFrequency := enactments.TrueConditionsFrequency()

		// put short names to them to make alghorithm easier to read
		failing := trueConditionsFrequency[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing]
		progressing := trueConditionsFrequency[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing]
		available := trueConditionsFrequency[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable]

		if failing > 0 {
			setPolicyFailedToConfigure(&m.policy.Status.Conditions, "TODO")
		} else if progressing > 0 {
			setPolicyProgressing(&m.policy.Status.Conditions, "TODO")
		} else if numberOfNodes > numberOfEnactments {
			setPolicyProgressing(&m.policy.Status.Conditions, "TODO")
		} else if available == numberOfNodes {
			setPolicySuccess(&m.policy.Status.Conditions, "TODO")
		} else {
			setPolicyNotMatching(&m.policy.Status.Conditions, "TODO")
		}
		err = m.client.Status().Update(context.TODO(), m.policy)
		if err != nil {
			return err
		}
		return nil
	})
}
