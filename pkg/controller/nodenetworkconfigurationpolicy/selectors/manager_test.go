package selectors

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller selectors", func() {
	var ExpectedNode = "node01"
	type nodeSelectorCase struct {
		ObjNodeName     string
		NodeSelector    map[string]string
		NodeLabels      map[string]string
		Matches         bool
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
					DesiredState: nmstatev1alpha1.State{Raw: nil}, // TODO
				},
			}

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			policySelectors := NewManager(fake.NewFakeClient(objs...), ExpectedNode, policy)

			policyUnmatchedLabels, policyMatches := policySelectors.MatchesThisNode()
			Expect(policyMatches).To(Equal(c.Matches))
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
				Matches: false,
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
				Matches:         true,
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
				Matches:         true,
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
				Matches:         true,
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
				Matches: false,
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
				Matches: false,
				UnmatchedLabels: map[string]string{
					"label2": "bar",
				},
			}),
		Entry("node not found",
			nodeSelectorCase{
				ObjNodeName:     "BadNode",
				NodeLabels:      map[string]string{},
				NodeSelector:    map[string]string{},
				Matches:         false,
				UnmatchedLabels: map[string]string{},
			}),
	)
})
