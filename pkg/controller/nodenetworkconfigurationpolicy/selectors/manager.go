package selectors

import (
	"context"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	manager.logger = logf.Log.WithName("policy/selectors/manager").WithValues("enactment", nmstatev1alpha1.EnactmentKey(nodeName, policy.Name).Name)
	return manager
}

func matches(nodeSelector map[string]string, labels map[string]string) (map[string]string, bool) {
	unmatchingLabels := map[string]string{}
	labelsMatches := true
	for key, value := range nodeSelector {
		if foundValue, hasKey := labels[key]; !hasKey || foundValue != value {
			unmatchingLabels[key] = value
			labelsMatches = false
		}
	}
	return unmatchingLabels, labelsMatches
}

func (m *Manager) MatchesThisNode() (map[string]string, bool) {
	node := corev1.Node{}
	err := m.client.Get(context.TODO(), types.NamespacedName{Name: m.nodeName}, &node)
	if err != nil {
		m.logger.Info("Cannot find corev1.Node", "nodeName", m.nodeName)
		return map[string]string{}, false
	}

	return matches(m.policy.Spec.NodeSelector, node.ObjectMeta.Labels)
}
