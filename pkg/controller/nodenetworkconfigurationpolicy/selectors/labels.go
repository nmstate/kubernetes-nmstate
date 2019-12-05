package selectors

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func matches(nodeSelector map[string]string, labels map[string]string) map[string]string {
	unmatchingLabels := map[string]string{}
	for key, value := range nodeSelector {
		if foundValue, hasKey := labels[key]; !hasKey || foundValue != value {
			unmatchingLabels[key] = value
		}
	}
	return unmatchingLabels
}

func (r *Request) UnmatchedLabels() (map[string]string, error) {
	node := corev1.Node{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: r.nodeName}, &node)
	if err != nil {
		r.logger.Info("Cannot find corev1.Node", "nodeName", r.nodeName)
		return map[string]string{}, err
	}

	return matches(r.policy.Spec.NodeSelector, node.ObjectMeta.Labels), nil
}
