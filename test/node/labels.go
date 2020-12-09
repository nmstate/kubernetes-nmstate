package node

import (
	"context"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

func AddLabels(nodeName string, labelsToAdd map[string]string) {
	node := corev1.Node{}
	err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving node to change labels")

	if len(node.Labels) == 0 {
		node.Labels = labelsToAdd
	} else {
		for k, v := range labelsToAdd {
			node.Labels[k] = v
		}
	}
	err = testenv.Client.Update(context.TODO(), &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success updating node with new labels")
}

func RemoveLabels(nodeName string, labelsToRemove map[string]string) {
	node := corev1.Node{}
	err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving node to remove labels")

	if len(node.Labels) == 0 {
		return
	}

	for k, _ := range labelsToRemove {
		delete(node.Labels, k)
	}

	err = testenv.Client.Update(context.TODO(), &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success updating node with label delete")
}
