package conditions

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Manager struct {
	client        client.Client
	policy        nmstatev1alpha1.NodeNetworkConfigurationPolicy
	enactmentName string
	logger        logr.Logger
}

func NewManager(client client.Client, nodeName string, policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) Manager {
	manager := Manager{
		client:        client,
		policy:        policy,
		enactmentName: fmt.Sprintf("%s-%s", nodeName, policy.Name),
	}
	manager.logger = logf.Log.WithName("policy/conditions/manager").WithValues("enactment", manager.enactmentName)
	return manager
}
func (m *Manager) NotifyNodeSelectorNotMatching(message string) {
	err := m.updateEnactmentCondition(setEnactmentNodeSelectorNotMatching, message)
	if err != nil {
		m.logger.Error(err, "Error notifying state NodeSelectorNotMatching")
	}
}
func (m *Manager) NotifyMatching() {
	err := m.updateEnactmentCondition(setEnactmentMatching, "All policy selectors are matching the node")
	if err != nil {
		m.logger.Error(err, "Error notifying state Matching")
	}
}
func (m *Manager) NotifyProgressing() {
	err := m.updateEnactmentCondition(setEnactmentProgressing, "Applying desired state")
	if err != nil {
		m.logger.Error(err, "Error notifying state Progressing")
	}
}
func (m *Manager) NotifyFailedToConfigure(failedErr error) {
	err := m.updateEnactmentCondition(setEnactmentFailedToConfigure, failedErr.Error())
	if err != nil {
		m.logger.Error(err, "Error notifying state FailingToConfigure")
	}
}

func (m *Manager) NotifySuccess() {
	err := m.updateEnactmentCondition(setEnactmentSuccess, "successfully reconciled")
	if err != nil {
		m.logger.Error(err, "Error notifying state Success")
	}
}

func (m *Manager) initializeEnactment() (*nmstatev1alpha1.NodeNetworkConfigurationEnactment, error) {
	//TODO: Don't harcode this take it from m.policy meta
	ownerRefList := []metav1.OwnerReference{{Name: m.policy.Name, Kind: "NodeNetworkConfigurationPolicy", APIVersion: "v1alpha1", UID: m.policy.UID}}

	enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{
		// Create NodeNetworkState for this node
		ObjectMeta: metav1.ObjectMeta{
			Name:            m.enactmentName,
			OwnerReferences: ownerRefList,
		},
	}

	err := m.client.Create(context.TODO(), &enactment)
	if err != nil {
		return nil, fmt.Errorf("error creating NodeNetworkConfigurationEnactment: %v, %+v", err, enactment)
	}

	//  We don't know yet at what phase we are
	enactment.Status.Phase = nmstatev1alpha1.NodeNetworkConfigurationEnactmentPhaseUnknown
	return &enactment, nil
}

func (m *Manager) updateEnactmentCondition(
	conditionsSetter func(*nmstatev1alpha1.NodeNetworkConfigurationEnactment, string),
	message string,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
		err := m.client.Get(context.TODO(), types.NamespacedName{Name: m.enactmentName}, instance)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			instance, err = m.initializeEnactment()
			if err != nil {
				return err
			}
		}
		conditionsSetter(instance, message)

		err = m.client.Status().Update(context.TODO(), instance)
		return err
	})
}
