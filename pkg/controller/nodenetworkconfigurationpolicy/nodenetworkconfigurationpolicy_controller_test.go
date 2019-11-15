package nodenetworkconfigurationpolicy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type predicateCase struct {
		ObjNodeName     string
		NodeSelector    map[string]string
		NodeLabels      map[string]string
		GenerationOld   int64
		GenerationNew   int64
		ReconcileCreate bool
		ReconcileUpdate bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   c.ObjNodeName,
					Labels: c.NodeLabels,
				},
			}

			oldNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationOld,
			}

			newNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationNew,
			}

			nodeNetworkConfigurationPolicy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
					NodeSelector: c.NodeSelector,
					DesiredState: nil, // TODO
				},
			}

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			cl := fake.NewFakeClient(objs...)
			predicate := forThisNodePredicate(cl)

			Expect(predicate.
				CreateFunc(event.CreateEvent{
					Meta:   &newNodeNetworkConfigurationPolicyMeta,
					Object: &nodeNetworkConfigurationPolicy,
				})).To(Equal(c.ReconcileCreate))
			Expect(predicate.
				UpdateFunc(event.UpdateEvent{
					MetaOld:   &oldNodeNetworkConfigurationPolicyMeta,
					ObjectOld: &nodeNetworkConfigurationPolicy,
					MetaNew:   &newNodeNetworkConfigurationPolicyMeta,
					ObjectNew: &nodeNetworkConfigurationPolicy,
				})).To(Equal(c.ReconcileUpdate))
			Expect(predicate.
				DeleteFunc(event.DeleteEvent{
					Meta:   &newNodeNetworkConfigurationPolicyMeta,
					Object: &nodeNetworkConfigurationPolicy,
				})).To(BeFalse())
		},
		Entry("events with empty node labels",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels:  map[string]string{},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: false,
				ReconcileUpdate: false,
			}),
		Entry("events with nil node selector",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:    nil,
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: true,
				ReconcileUpdate: true,
			}),
		Entry("events with empty node selector",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector:    map[string]string{},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: true,
				ReconcileUpdate: true,
			}),
		Entry("events with matching node selector",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: true,
				ReconcileUpdate: true,
			}),
		Entry("events with missing label at node",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: false,
				ReconcileUpdate: false,
			}),
		Entry("events with different label value at node",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar1",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: false,
				ReconcileUpdate: false,
			}),
		Entry("node not found",
			predicateCase{
				ObjNodeName:     "node02",
				NodeLabels:      map[string]string{},
				NodeSelector:    map[string]string{},
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: false,
				ReconcileUpdate: false,
			}),
		Entry("generation remains the same",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				GenerationOld:   1,
				GenerationNew:   1,
				ReconcileCreate: true,
				ReconcileUpdate: false,
			}),
	)
})
