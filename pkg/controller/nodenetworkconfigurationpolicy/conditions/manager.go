package conditions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Manager struct {
	client   client.Client
	policy   nmstatev1alpha1.NodeNetworkConfigurationPolicy
	nodeName string
	logger   logr.Logger
}

func NewManager(client client.Client, nodeName string, policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) Manager {
	manager := Manager{
		client:   client,
		policy:   policy,
		nodeName: nodeName,
	}
	manager.logger = logf.Log.WithName("policy/conditions/manager").WithValues("enactment", nmstatev1alpha1.EnactmentKey(nodeName, policy.Name).Name)
	return manager
}
func (m *Manager) NotifyNodeSelectorNotMatching(unmatchingLabels map[string]string) {
	message := ""
	if len(unmatchingLabels) > 0 {
		message = fmt.Sprintf("Unmatching labels: %v", unmatchingLabels)
	} else {
		message = fmt.Sprintf("Cannot retrieve node %d", m.nodeName)
	}
	err := m.updateEnactmentConditions(setEnactmentNodeSelectorNotMatching, message)
	if err != nil {
		m.logger.Error(err, "Error notifying state NodeSelectorNotMatching")
	}
}
func (m *Manager) NotifyMatching() {
	err := m.updateEnactmentConditions(setEnactmentMatching, "All policy selectors are matching the node")
	if err != nil {
		m.logger.Error(err, "Error notifying state Matching")
	}
}
func (m *Manager) NotifyProgressing() {
	err := m.updateEnactmentConditions(setEnactmentProgressing, "Applying desired state")
	if err != nil {
		m.logger.Error(err, "Error notifying state Progressing")
	}
}
func (m *Manager) NotifyFailedToConfigure(failedErr error) {
	err := m.updateEnactmentConditions(setEnactmentFailedToConfigure, failedErr.Error())
	if err != nil {
		m.logger.Error(err, "Error notifying state FailingToConfigure")
	}
}

func (m *Manager) NotifySuccess() {
	err := m.updateEnactmentConditions(setEnactmentSuccess, "successfully reconciled")
	if err != nil {
		m.logger.Error(err, "Error notifying state Success")
	}
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
