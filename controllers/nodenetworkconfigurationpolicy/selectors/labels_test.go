package selectors

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
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

			policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{
				Spec: nmstate.NodeNetworkConfigurationPolicySpec{
					NodeSelector: c.NodeSelector,
				},
			}

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			selectorsRequest := NewFromPolicy(fake.NewFakeClient(objs...), policy)

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
