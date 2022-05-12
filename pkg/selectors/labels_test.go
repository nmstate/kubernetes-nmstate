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

package selectors

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller selectors", func() {
	var expectedNode = "node01"
	type nodeSelectorCase struct {
		ObjNodeName         string
		NodeSelector        map[string]string
		NodeLabels          map[string]string
		MatchResult         types.GomegaMatcher
		UnmatchedNodeLabels map[string]string
	}
	DescribeTable("testing node selectors",
		func(c nodeSelectorCase) {
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   c.ObjNodeName,
					Labels: c.NodeLabels,
				},
			}

			policy := nmstatev1.NodeNetworkConfigurationPolicy{
				Spec: nmstate.NodeNetworkConfigurationPolicySpec{
					NodeSelector: c.NodeSelector,
				},
			}

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			fakeClient := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()
			selectorsRequest := NewFromPolicy(fakeClient, &policy)

			unmatchedNodeLabels, err := selectorsRequest.UnmatchedNodeLabels(expectedNode)
			Expect(err).To(c.MatchResult)
			Expect(unmatchedNodeLabels).To(Equal(c.UnmatchedNodeLabels))

		},
		Entry("events with empty node labels",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels:  map[string]string{},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedNodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
			}),
		Entry("events with nil node selector",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:        nil,
				MatchResult:         Succeed(),
				UnmatchedNodeLabels: map[string]string{},
			}),
		Entry("events with empty node selector",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:        map[string]string{},
				MatchResult:         Succeed(),
				UnmatchedNodeLabels: map[string]string{},
			}),
		Entry("events with matching node selector",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult:         Succeed(),
				UnmatchedNodeLabels: map[string]string{},
			}),
		Entry("events with missing label at node",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedNodeLabels: map[string]string{
					"label2": "bar",
				},
			}),
		Entry("events with different label value at node",
			nodeSelectorCase{
				ObjNodeName: expectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar1",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedNodeLabels: map[string]string{
					"label2": "bar",
				},
			}),
		Entry("node not found",
			nodeSelectorCase{
				ObjNodeName:         "BadNode",
				NodeLabels:          map[string]string{},
				NodeSelector:        map[string]string{},
				MatchResult:         MatchError(apierrors.NewNotFound(schema.GroupResource{Resource: "nodes"}, expectedNode)),
				UnmatchedNodeLabels: map[string]string{},
			}),
	)
})
