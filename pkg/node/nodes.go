package node

import (
	"context"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactment"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
)

const (
	DEFAULT_MAXUNAVAILABLE = "50%"
)

func NodesRunningNmstate(cli client.Client, nodeSelector map[string]string) ([]corev1.Node, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), &nodes, client.MatchingLabels(nodeSelector))
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

func MaxUnavailableNodeCount(cli client.Client, policy *nmstatev1beta1.NodeNetworkConfigurationPolicy) (int, error) {
	enactmentsTotal, _, err := enactment.CountByPolicy(cli, policy)
	if err != nil {
		return 0, err
	}
	intOrPercent := intstr.FromString(DEFAULT_MAXUNAVAILABLE)
	if policy.Spec.MaxUnavailable != nil {
		intOrPercent = *policy.Spec.MaxUnavailable
	}
	maxUnavailable, err := ScaledMaxUnavailableNodeCount(enactmentsTotal, intOrPercent)
	return maxUnavailable, nil
}

func ScaledMaxUnavailableNodeCount(matchingNodes int, intOrPercent intstr.IntOrString) (int, error) {
	maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(&intOrPercent, matchingNodes, true)
	if err != nil {
		return 0, err
	}
	if maxUnavailable < 1 {
		maxUnavailable = 1
	}
	return maxUnavailable, nil
}
