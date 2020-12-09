package nodenetworkconfigurationpolicy

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

func onModifiedNodeLabels(oldNode, newNode corev1.Node) bool {
	return !reflect.DeepEqual(oldNode.Labels, newNode.Labels)
}
