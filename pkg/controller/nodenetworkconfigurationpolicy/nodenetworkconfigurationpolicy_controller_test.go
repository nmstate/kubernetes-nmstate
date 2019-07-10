package nodenetworkconfigurationpolicy

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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

const (
	RECONCILE_POLICY_NAME = "nodenetworkconfigurationpolicy-test"
	NUMBER_OF_NODES       = 2
)

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type predicateCase struct {
		ObjNodeName  string
		NodeSelector map[string]string
		NodeLabels   map[string]string
		Reconcile    bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
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
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels:  map[string]string{},
				NodeSelector: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				Reconcile: false,
			}),
		Entry("events with nil node selector",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: nil,
				Reconcile:    true,
			}),
		Entry("events with empty node selector",
			predicateCase{
				ObjNodeName: "node01",
				NodeLabels: map[string]string{
					"label1": "foo",
					"label2": "bar",
				},
				NodeSelector: map[string]string{},
				Reconcile:    true,
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
				Reconcile: true,
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
				Reconcile: false,
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
				Reconcile: false,
			}),
		Entry("node not found",
			predicateCase{
				ObjNodeName:  "node02",
				NodeLabels:   map[string]string{},
				NodeSelector: map[string]string{},
				Reconcile:    false,
			}),
	)
})

var _ = Describe("NodeNetworkConfigurationPolicy controller reconciler", func() {
	var (
		runningNode       = corev1.Node{}
		runningNodeName   string
		nodeNetworkStates []nmstatev1alpha1.NodeNetworkState

		cl               client.Client
		r                *ReconcileNodeNetworkConfigurationPolicy
		req              reconcile.Request
		reconciledPolicy = nmstatev1alpha1.NodeNetworkConfigurationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: RECONCILE_POLICY_NAME,
			},
		}
		eth0UpState = nmstatev1alpha1.State(`
interfaces:
  - name: eth0
    state: up
`)
		eth1UpState = nmstatev1alpha1.State(`
interfaces:
  - name: eth1
    state: up
`)
		eth0MTU1450 = nmstatev1alpha1.State(`
interfaces:
  - name: eth0
    state: up
    mtu: 1450
`)
		eth0AndEth1UpState = nmstatev1alpha1.State(`
interfaces:
  - name: eth0
    state: up
  - name: eth1
    state: up
`)
	)

	BeforeEach(func() {

		By("populate the NodeNetworkState for each node")
		nodeNetworkStates = nil
		for n := 1; n <= NUMBER_OF_NODES; n++ {
			nodeName := fmt.Sprintf("node%02d", n)
			nodeNetworkStates = append(nodeNetworkStates, nmstatev1alpha1.NodeNetworkState{
				ObjectMeta: metav1.ObjectMeta{
					Name:   nodeName,
					Labels: map[string]string{"hostname": nodeName},
				},
			})
		}

		// Mock request to simulate Reconcile() being called on an event for a
		// watched resource .
		req = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: RECONCILE_POLICY_NAME,
			},
		}

		runningNode.ObjectMeta.Name = nodeNetworkStates[0].Name
		runningNodeName = runningNode.ObjectMeta.Name
		runningNode.ObjectMeta.Labels = map[string]string{"hostname": runningNodeName}

		By("set reconciling policy labels with hostname")
		reconciledPolicy.Spec.NodeSelector = map[string]string{"hostname": runningNodeName}

	})

	JustBeforeEach(func() {
		// Register operator types with the runtime scheme.
		By("register state and policies types")
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &nmstatev1alpha1.NodeNetworkState{})
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &nmstatev1alpha1.NodeNetworkConfigurationPolicy{})
		s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &nmstatev1alpha1.NodeNetworkConfigurationPolicyList{})

		// Objects to track in the fake client
		objs := []runtime.Object{&reconciledPolicy}

		By("create fake client")
		cl = fake.NewFakeClient(objs...)

		r = &ReconcileNodeNetworkConfigurationPolicy{client: cl, scheme: s}
	})

	Context("when there is no NodeNetworkState for the node", func() {
		It("should requeue", func() {
			res, err := r.Reconcile(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Requeue).To(BeTrue())
		})
	})

	Context("when there is NodeNetworkState for the node", func() {

		JustBeforeEach(func() {
			err := cl.Create(context.TODO(), &nodeNetworkStates[0])
			Expect(err).ToNot(HaveOccurred())
			err = cl.Create(context.TODO(), &runningNode)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not requeue", func() {
			res, err := r.Reconcile(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Requeue).ToNot(BeTrue())
		})
		Context(" and it has non empty desiredState", func() {
			BeforeEach(func() {
				reconciledPolicy.Spec.DesiredState = eth0UpState
			})

			It("should update NodeNetworkState with desiredState", func() {
				_, err := r.Reconcile(req)
				Expect(err).ToNot(HaveOccurred())
				obtainedState := nmstatev1alpha1.NodeNetworkState{}
				err = cl.Get(context.TODO(), types.NamespacedName{Name: runningNodeName}, &obtainedState)
				Expect(err).ToNot(HaveOccurred())
				Expect(obtainedState.Spec.DesiredState).To(MatchYAML(eth0UpState))
			})
			Context(" and has another policy for different node", func() {
				JustBeforeEach(func() {
					differentNodePolicy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
						ObjectMeta: metav1.ObjectMeta{
							Name: "different-node-policy",
						},
						Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
							NodeSelector: map[string]string{
								"hostname": "arrakis",
							},
							DesiredState: eth1UpState,
						},
					}
					err := cl.Create(context.TODO(), &differentNodePolicy)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should not merge desired state", func() {
					_, err := r.Reconcile(req)
					Expect(err).ToNot(HaveOccurred())
					obtainedState := nmstatev1alpha1.NodeNetworkState{}
					err = cl.Get(context.TODO(), types.NamespacedName{Name: runningNodeName}, &obtainedState)
					Expect(err).ToNot(HaveOccurred())
					Expect(obtainedState.Spec.DesiredState).To(MatchYAML(eth0UpState))
				})
			})
			Context(" and has another policy with non conflicting desiredState", func() {
				JustBeforeEach(func() {
					sameNodePolicy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
						ObjectMeta: metav1.ObjectMeta{
							Name: "same-node-policy",
						},
						Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
							NodeSelector: map[string]string{
								"hostname": runningNodeName,
							},
							DesiredState: eth1UpState,
						},
					}
					err := cl.Create(context.TODO(), &sameNodePolicy)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should merge desired state", func() {
					_, err := r.Reconcile(req)
					Expect(err).ToNot(HaveOccurred())
					obtainedState := nmstatev1alpha1.NodeNetworkState{}
					err = cl.Get(context.TODO(), types.NamespacedName{Name: runningNodeName}, &obtainedState)
					Expect(err).ToNot(HaveOccurred())
					Expect(obtainedState.Spec.DesiredState).To(MatchYAML(eth0AndEth1UpState))
				})
			})
			Context(" and has another policy with conflicting desiredState", func() {
				JustBeforeEach(func() {
					conflictingPolicy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
						ObjectMeta: metav1.ObjectMeta{
							Name: "conflicting-policy",
						},
						Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
							NodeSelector: map[string]string{
								"hostname": runningNodeName,
							},
							DesiredState: eth0MTU1450,
						},
					}
					err := cl.Create(context.TODO(), &conflictingPolicy)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should keep desired state", func() {
					res, err := r.Reconcile(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.Requeue).To(BeFalse())
					obtainedState := nmstatev1alpha1.NodeNetworkState{}
					err = cl.Get(context.TODO(), types.NamespacedName{Name: runningNodeName}, &obtainedState)
					Expect(err).ToNot(HaveOccurred())
					Expect(obtainedState.Spec.DesiredState).To(MatchYAML(eth0UpState))
				})
			})
		})
	})
})
