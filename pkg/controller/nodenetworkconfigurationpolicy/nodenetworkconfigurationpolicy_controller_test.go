package nodenetworkconfigurationpolicy

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type PredicateCase struct {
		EnvNodeName  string
		ObjNodeName  string
		NodeSelector map[string]string
		NodeLabels   map[string]string
		Reconcile    bool
	}
	DescribeTable("all events",
		func(c PredicateCase) {
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   c.ObjNodeName,
					Labels: c.NodeLabels,
				},
			}

			nodeNetworkConfigurationPolicy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
					NodeSelector: c.NodeSelector,
				},
			}

			os.Setenv("NODE_NAME", c.EnvNodeName)

			// Objects to track in the fake client
			objs := []runtime.Object{&node}
			cl := fake.NewFakeClient(objs...)
			predicate := forThisNodePredicate(cl)

			Expect(predicate.
				CreateFunc(event.CreateEvent{
					Object: &nodeNetworkConfigurationPolicy})).To(Equal(c.Reconcile))
			Expect(predicate.
				DeleteFunc(event.DeleteEvent{
					Object: &nodeNetworkConfigurationPolicy})).To(Equal(c.Reconcile))
			Expect(predicate.
				GenericFunc(event.GenericEvent{
					Object: &nodeNetworkConfigurationPolicy})).To(Equal(c.Reconcile))
			Expect(predicate.
				UpdateFunc(event.UpdateEvent{
					ObjectOld: &nodeNetworkConfigurationPolicy,
					ObjectNew: &nodeNetworkConfigurationPolicy,
				})).To(Equal(c.Reconcile))
		},
		Entry("events with empty node labels",
			PredicateCase{
				EnvNodeName: "node01",
				ObjNodeName: "node01",
				NodeLabels:  map[string]string{},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				Reconcile: false,
			}),
		Entry("events with empty node selector",
			PredicateCase{
				ObjNodeName: "node01",
				EnvNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{},
				Reconcile:    true,
			}),
		Entry("events with matching node selector",
			PredicateCase{
				ObjNodeName: "node01",
				EnvNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				Reconcile: true,
			}),
		Entry("events with missing label at node",
			PredicateCase{
				ObjNodeName: "node01",
				EnvNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				Reconcile: false,
			}),
		Entry("events with different label value at node",
			PredicateCase{
				ObjNodeName: "node01",
				EnvNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar1",
				},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				Reconcile: false,
			}),
		Entry("node not found",
			PredicateCase{
				ObjNodeName:  "node01",
				EnvNodeName:  "node02",
				NodeLabels:   map[string]string{},
				NodeSelector: map[string]string{},
				Reconcile:    false,
			}),
		Entry("env NODE_NAME empty",
			PredicateCase{
				EnvNodeName:  "",
				ObjNodeName:  "node01",
				NodeLabels:   map[string]string{},
				NodeSelector: map[string]string{},
				Reconcile:    false,
			}),
	)
})

var _ = Describe("NodeNetworkConfigurationPolicy controller reconciler", func() {
	var (
		name          = "nodenetworkconfigurationpolicy-test"
		namespace     = "default"
		numberOfNodes = 2
		nodes         []nmstatev1alpha1.NodeNetworkState

		cl      client.Client
		r       *ReconcileNodeNetworkConfigurationPolicy
		req     reconcile.Request
		policy1 = nmstatev1alpha1.NodeNetworkConfigurationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		res reconcile.Result
	)

	BeforeEach(func() {

		By("populate the NodeNetworkState for nodes")
		nodes = nil
		for n := 1; n <= numberOfNodes; n++ {
			nodes = append(nodes, nmstatev1alpha1.NodeNetworkState{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("node%02d", n),
					Namespace: namespace,
				},
			})
		}

		// Mock request to simulate Reconcile() being called on an event for a
		// watched resource .
		req = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
		By("set NODE_NAME to " + nodes[0].Name)
		os.Setenv("NODE_NAME", nodes[0].Name)
	})

	JustBeforeEach(func() {

		// Register operator types with the runtime scheme.
		By("register state and policies types")
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &policy1)
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &nodes[0])

		// Objects to track in the fake client
		objs := []runtime.Object{&policy1}

		By("create fake client")
		cl = fake.NewFakeClient(objs...)

		r = &ReconcileNodeNetworkConfigurationPolicy{client: cl, scheme: s}

	})
	Context("when there is no NODE_NAME environment variable", func() {
		BeforeEach(func() {
			os.Unsetenv("NODE_NAME")
		})
		It("should fail", func() {
			_, err := r.Reconcile(req)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when NODE_NAME environment variable is empty", func() {
		BeforeEach(func() {
			os.Setenv("NODE_NAME", "")
		})
		It("should fail", func() {
			_, err := r.Reconcile(req)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when there is no NodeNetworkState for the node", func() {
		JustBeforeEach(func() {
			var err error
			res, err = r.Reconcile(req)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should requeue", func() {
			Expect(res.Requeue).To(BeTrue())
		})
	})
	Context("when there is NodeNetworkState for the node", func() {
		JustBeforeEach(func() {
			err := cl.Create(context.TODO(), &nodes[0])
			Expect(err).ToNot(HaveOccurred())
			res, err = r.Reconcile(req)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not requeue", func() {
			Expect(res.Requeue).ToNot(BeTrue())
		})
		Context(" and it has non empty desiredState", func() {
			var (
				expectedDesiredState = nmstatev1alpha1.State(`
interfaces:
  name: eth0
  state: up
`)
			)
			BeforeEach(func() {
				policy1.Spec.DesiredState = expectedDesiredState
			})
			It("should update NodeNetworkState with desiredState", func() {
				obtainedState := nmstatev1alpha1.NodeNetworkState{}
				err := cl.Get(context.TODO(), types.NamespacedName{Name: nodes[0].Name}, &obtainedState)
				Expect(err).ToNot(HaveOccurred())
				Expect(obtainedState.Spec.DesiredState).To(MatchYAML(expectedDesiredState))
			})
		})
	})
})
