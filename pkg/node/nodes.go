/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"context"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactment"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
)

const (
	DefaultMaxunavailable = "50%"
	MinMaxunavailable     = 1
)

type MaxUnavailableLimitReachedError struct{}

func (f MaxUnavailableLimitReachedError) Error() string {
	return "maximal number of nodes are already processing policy configuration"
}

func NodesRunningNmstate(cli client.Reader, nodeSelector map[string]string) ([]corev1.Node, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), &nodes, client.MatchingLabels(nodeSelector))
	if err != nil {
		return []corev1.Node{}, errors.Wrap(err, "getting nodes failed")
	}

	pods := corev1.PodList{}
	byComponent := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
	err = cli.List(context.TODO(), &pods, byComponent)
	if err != nil {
		return []corev1.Node{}, errors.Wrap(err, "getting pods failed")
	}

	filteredNodes := []corev1.Node{}
	for nodeIndex := range nodes.Items {
		for podIndex := range pods.Items {
			if nodes.Items[nodeIndex].Name == pods.Items[podIndex].Spec.NodeName {
				filteredNodes = append(filteredNodes, nodes.Items[nodeIndex])
				break
			}
		}
	}
	return filteredNodes, nil
}

func MaxUnavailableNodeCount(cli client.Reader, policy *nmstatev1.NodeNetworkConfigurationPolicy) (int, error) {
	enactmentsTotal, _, err := enactment.CountByPolicy(cli, policy)
	if err != nil {
		return MinMaxunavailable, err
	}
	intOrPercent := intstr.FromString(DefaultMaxunavailable)
	if policy.Spec.MaxUnavailable != nil {
		intOrPercent = *policy.Spec.MaxUnavailable
	}
	return ScaledMaxUnavailableNodeCount(enactmentsTotal, intOrPercent)
}

func ScaledMaxUnavailableNodeCount(matchingNodes int, intOrPercent intstr.IntOrString) (int, error) {
	correctMaxUnavailable := func(maxUnavailable int) int {
		if maxUnavailable < 1 {
			return MinMaxunavailable
		}
		return maxUnavailable
	}
	maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(&intOrPercent, matchingNodes, true)
	if err != nil {
		defaultMaxUnavailable := intstr.FromString(DefaultMaxunavailable)
		maxUnavailable, _ = intstr.GetScaledValueFromIntOrPercent(
			&defaultMaxUnavailable,
			matchingNodes,
			true,
		)
		return correctMaxUnavailable(maxUnavailable), err
	}
	return correctMaxUnavailable(maxUnavailable), nil
}
