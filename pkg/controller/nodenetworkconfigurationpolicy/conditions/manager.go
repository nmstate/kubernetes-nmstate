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

var (
	log = logf.Log.WithName("policy/conditions/manager")
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
		logger:     log.WithValues("node", nodeName, "policy", policyName),
	}
}

func (m *Manager) Progressing() {
	err := m.updateEnactmentCondition(setEnactmentProgressing, "Applying desired state")
	if err != nil {
		m.logger.Error(err, "Error change state to progressing")
	}
}
func (m *Manager) FailedToConfigure(failedErr error) {
	err := m.updateEnactmentCondition(setEnactmentFailedToConfigure, failedErr.Error())
	if err != nil {
		m.logger.Error(err, "Error chaing state to failing at configure with error: %v", failedErr)
	}
}

func (m *Manager) FailedToFindPolicy(failedErr error) {
	err := m.updateEnactmentCondition(setEnactmentFailedToFindPolicy, failedErr.Error())
	if err != nil {
		m.logger.Error(err, "Error changing state to finling at finding policy with error: %v", failedErr)
	}
}

func (m *Manager) Success() {
	err := m.updateEnactmentCondition(setEnactmentSuccess, "successfully reconciled")
	if err != nil {
		m.logger.Error(err, "Success condition update failed while reporting success: %v", err)
	}
}

func (m *Manager) updateEnactmentCondition(
	condition func(*nmstatev1alpha1.EnactmentList, string, string),
	message string,
) error {
	// Set enactment condition
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1alpha1.NodeNetworkState{}
		err := m.client.Get(context.TODO(), types.NamespacedName{Name: m.nodeName}, instance)
		if err != nil {
			return err
		}

		condition(&instance.Status.Enactments, m.policyName.Name, message)

		err = m.client.Status().Update(context.TODO(), instance)
		return err
	})
}
