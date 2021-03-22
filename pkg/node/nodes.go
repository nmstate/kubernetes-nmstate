package node

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
)

func NodesRunningNmstate(cli client.Client) ([]corev1.Node, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), &nodes)
	if err != nil {
		return []corev1.Node{}, errors.Wrap(err, "getting nodes failed")
	}

	pods := corev1.PodList{}
	byApp := client.MatchingLabels{"app": "kubernetes-nmstate"}
	err = cli.List(context.TODO(), &pods, byApp)
	if err != nil {
		return []corev1.Node{}, errors.Wrap(err, "getting pods failed")
	}

	filteredNodes := []corev1.Node{}
	for _, node := range nodes.Items {
		for _, pod := range pods.Items {
			if node.Name == pod.Spec.NodeName {
				filteredNodes = append(filteredNodes, node)
				break
			}
		}
	}
	return filteredNodes, nil
}
