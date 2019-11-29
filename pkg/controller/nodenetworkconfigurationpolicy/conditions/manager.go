package conditions

import (
	"context"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Manager struct {
	client     client.Client
	nodeName   string
	policyName types.NamespacedName
	logger     logr.Logger
}

func NewManager(client client.Client, nodeName string, policyName types.NamespacedName) Manager {
	return Manager{
		client:     client,
		nodeName:   nodeName,
		policyName: policyName,
		logger:     logf.Log.WithName("policy/conditions/manager").WithValues("node", nodeName, "policy", policyName),
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

func (m *Manager) updateEnactmentCondition(
	conditionsSetter func(*nmstatev1alpha1.EnactmentList, string, string),
	message string,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
		err := m.client.Get(context.TODO(), m.policyName, instance)
		if err != nil {
			return err
		}

		conditionsSetter(&instance.Status.Enactments, m.nodeName, message)

		err = m.client.Status().Update(context.TODO(), instance)
		return err
	})
}
