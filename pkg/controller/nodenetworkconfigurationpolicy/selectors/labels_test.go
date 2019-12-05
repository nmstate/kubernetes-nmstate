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

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller selectors", func() {
	var ExpectedNode = "node01"
	type nodeSelectorCase struct {
		ObjNodeName     string
		NodeSelector    map[string]string
		NodeLabels      map[string]string
		MatchResult     types.GomegaMatcher
		UnmatchedLabels map[string]string
	}
	DescribeTable("testing node selectors",
		func(c nodeSelectorCase) {
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   c.ObjNodeName,
					Labels: c.NodeLabels,
				},
			}

			policy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
					NodeSelector: c.NodeSelector,
				},
			}

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			policySelectors := New(fake.NewFakeClient(objs...), ExpectedNode, policy)

			policyUnmatchedLabels, err := policySelectors.UnmatchedLabels()
			Expect(err).To(c.MatchResult)
			Expect(policyUnmatchedLabels).To(Equal(c.UnmatchedLabels))

		},
		Entry("events with empty node labels",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels:  map[string]string{},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
			}),
		Entry("events with nil node selector",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:    nil,
				MatchResult:     Succeed(),
				UnmatchedLabels: map[string]string{},
			}),
		Entry("events with empty node selector",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:    map[string]string{},
				MatchResult:     Succeed(),
				UnmatchedLabels: map[string]string{},
			}),
		Entry("events with matching node selector",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult:     Succeed(),
				UnmatchedLabels: map[string]string{},
			}),
		Entry("events with missing label at node",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedLabels: map[string]string{
					"label2": "bar",
				},
			}),
		Entry("events with different label value at node",
			nodeSelectorCase{
				ObjNodeName: ExpectedNode,
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar1",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				MatchResult: Succeed(),
				UnmatchedLabels: map[string]string{
					"label2": "bar",
				},
			}),
		Entry("node not found",
			nodeSelectorCase{
				ObjNodeName:     "BadNode",
				NodeLabels:      map[string]string{},
				NodeSelector:    map[string]string{},
				MatchResult:     MatchError(apierrors.NewNotFound(schema.GroupResource{Resource: "nodes"}, ExpectedNode)),
				UnmatchedLabels: map[string]string{},
			}),
	)
})
