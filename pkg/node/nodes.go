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
	"fmt"
	"strconv"
	"strings"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactment"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// InvalidMaxUnavailableError indicates that a policy's maxUnavailable resolves to a
// non-positive number of nodes. This should normally be rejected at admission time by
// the CRD CEL rule, but it can still be observed for policies persisted before the CRD
// was upgraded. It is a terminal validation error: reconciliation must fail the
// enactment rather than retry, since the value can only be fixed by editing the policy.
type InvalidMaxUnavailableError struct {
	value string
}

func (e InvalidMaxUnavailableError) Error() string {
	return fmt.Sprintf("maxUnavailable must be a positive integer or a percentage between 1%% and 100%%, got %q", e.value)
}

func NodesRunningNmstate(ctx context.Context, cli client.Reader, nodeSelector map[string]string) ([]corev1.Node, error) {
	nodes := corev1.NodeList{}
	err := cli.List(ctx, &nodes, client.MatchingLabels(nodeSelector))
	if err != nil {
		return []corev1.Node{}, errors.Wrap(err, "getting nodes failed")
	}

	pods := corev1.PodList{}
	byComponent := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
	handlerNamespace := environment.GetEnvVar("POD_NAMESPACE", "")
	if handlerNamespace == "" {
		handlerNamespace = environment.GetEnvVar("HANDLER_NAMESPACE", "")
	}
	if handlerNamespace == "" {
		return []corev1.Node{}, errors.New("failed to get nodes running nmstate: neither POD_NAMESPACE nor HANDLER_NAMESPACE is set")
	}
	err = cli.List(ctx, &pods, byComponent, client.InNamespace(handlerNamespace))
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

func FilterReady(nodes []corev1.Node) []corev1.Node {
	filteredNodes := []corev1.Node{}
	for i := range nodes {
		if isReady(&nodes[i]) {
			filteredNodes = append(filteredNodes, nodes[i])
		}
	}
	return filteredNodes
}

func isReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func MaxUnavailableNodeCount(ctx context.Context, cli client.Reader, policy *nmstatev1.NodeNetworkConfigurationPolicy) (int, error) {
	enactmentsTotal, _, err := enactment.CountByPolicy(ctx, cli, policy)
	if err != nil {
		return MinMaxunavailable, err
	}
	intOrPercent := intstr.FromString(DefaultMaxunavailable)
	if policy.Spec.MaxUnavailable != nil {
		intOrPercent = *policy.Spec.MaxUnavailable
	}
	return ScaledMaxUnavailableNodeCount(enactmentsTotal, intOrPercent)
}

// scaledDefaultMaxUnavailableNodeCount returns the DefaultMaxunavailable percentage
// scaled to the given number of matching nodes. Used as the fallback count when the
// configured value cannot be resolved.
func scaledDefaultMaxUnavailableNodeCount(matchingNodes int) int {
	defaultMaxUnavailable := intstr.FromString(DefaultMaxunavailable)
	maxUnavailable, _ := intstr.GetScaledValueFromIntOrPercent(&defaultMaxUnavailable, matchingNodes, true)
	return maxUnavailable
}

func ScaledMaxUnavailableNodeCount(matchingNodes int, intOrPercent intstr.IntOrString) (int, error) {
	// GetScaledValueFromIntOrPercent only accepts int values and strings ending in
	// "%"; it rejects a string that holds a plain integer (e.g. "5"). Treat such a
	// string as an absolute node count so both "5" and "5%" are accepted. The CRD
	// CEL rule only admits [1-9][0-9]* for this form, so a parse failure means the
	// value is either out of int range or bypassed admission (e.g. persisted before
	// the CRD was upgraded); either way it is terminally invalid rather than a value
	// that should silently fall back to the default.
	if intOrPercent.Type == intstr.String && !strings.HasSuffix(intOrPercent.StrVal, "%") {
		v, convErr := strconv.Atoi(intOrPercent.StrVal)
		if convErr != nil {
			return scaledDefaultMaxUnavailableNodeCount(matchingNodes), InvalidMaxUnavailableError{value: intOrPercent.String()}
		}
		intOrPercent = intstr.FromInt(v)
	}
	maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(&intOrPercent, matchingNodes, true)
	if err != nil {
		return scaledDefaultMaxUnavailableNodeCount(matchingNodes), err
	}
	// Defense-in-depth: the CRD CEL rule rejects non-positive maxUnavailable values on
	// write, but it only runs on create/update and cannot catch policies persisted
	// before the CRD was upgraded. A value below MinMaxunavailable would make
	// incrementUnavailableNodeCount treat the limit as reached on the first node and
	// deadlock the rollout, so fail validation instead. matchingNodes is guarded so a
	// valid policy that simply matches no nodes is not reported as invalid.
	if matchingNodes > 0 && maxUnavailable < MinMaxunavailable {
		return maxUnavailable, InvalidMaxUnavailableError{value: intOrPercent.String()}
	}
	return maxUnavailable, nil
}

// Return true if the event name is the name of
// the pods's node (reading the env var NODE_NAME)
func EventIsForThisNode(meta v1.Object) bool {
	createdNodeName := meta.GetName()
	podNodeName := environment.NodeName()
	// Only reconcile is it's for this pod
	return createdNodeName == podNodeName
}
