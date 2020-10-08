package selectors

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func unmatchingLabels(nodeSelector map[string]string, labels map[string]string) map[string]string {
	unmatchingLabels := map[string]string{}
	for key, value := range nodeSelector {
		if foundValue, hasKey := labels[key]; !hasKey || foundValue != value {
			unmatchingLabels[key] = value
		}
	}
	return unmatchingLabels
}

func (s *Selectors) UnmatchedNodeLabels(nodeName string) (map[string]string, error) {
	logger := s.logger.WithValues("node", nodeName)
	node := corev1.Node{}
	err := s.client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	if err != nil {
		logger.Info("Cannot find corev1.Node")
		return map[string]string{}, err
	}

	return unmatchingLabels(s.policy.Spec.NodeSelector, node.ObjectMeta.Labels), nil
}
