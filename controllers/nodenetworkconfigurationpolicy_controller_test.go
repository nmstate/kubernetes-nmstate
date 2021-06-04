package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller", func() {
	type predicateCase struct {
		GenerationOld   int64
		GenerationNew   int64
		ReconcileCreate bool
		ReconcileUpdate bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
			oldNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationOld,
			}

			newNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationNew,
			}

			nodeNetworkConfigurationPolicy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}

			predicate := onCreateOrUpdateWithDifferentGeneration

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
		Entry("generation remains the same",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   1,
				ReconcileCreate: true,
				ReconcileUpdate: false,
			}),
		Entry("generation is different",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: true,
				ReconcileUpdate: true,
			}),
	)
	type claimNodeRunningUpdateCase struct {
		currentNodeRunningUpdate  string
		expectedNodeRunningUpdate string
		shouldConflict            bool
	}
	DescribeTable("when claimNodeRunningUpdate is called and",
		func(c claimNodeRunningUpdateCase) {
			reconciler := NodeNetworkConfigurationPolicyReconciler{}
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.GroupVersion,
				&nmstatev1beta1.NodeNetworkConfigurationPolicy{},
			)

			nncp := nmstatev1beta1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Status: shared.NodeNetworkConfigurationPolicyStatus{
					NodeRunningUpdate: c.currentNodeRunningUpdate,
				},
			}

			objs := []runtime.Object{&nncp}

			// Create a fake client to mock API calls.
			cl := fake.NewFakeClientWithScheme(s, objs...)

			reconciler.Client = cl
			reconciler.Log = ctrl.Log.WithName("controllers").WithName("NodeNetworkConfigurationPolicy")

			err := reconciler.claimNodeRunningUpdate(&nncp)
			if c.shouldConflict {
				Expect(err).Should(WithTransform(apierrors.IsConflict, BeTrue()), "should conflict")
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
			obtainedNNCP := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
			cl.Get(context.TODO(), types.NamespacedName{Name: nncp.Name}, &obtainedNNCP)
			Expect(obtainedNNCP.Status.NodeRunningUpdate).To(Equal(c.expectedNodeRunningUpdate))
		},
		Entry("there is no node configuring network, should update nodeRunnigUpdate field",
			claimNodeRunningUpdateCase{
				expectedNodeRunningUpdate: nodeName,
				shouldConflict:            false,
			}),
		Entry("there is different node configuring network, should fail with conflict",
			claimNodeRunningUpdateCase{
				currentNodeRunningUpdate:  nodeName + "foo",
				expectedNodeRunningUpdate: nodeName + "foo",
				shouldConflict:            true,
			}),
		Entry("the node running the handler is configuring the network, should not conflict",
			claimNodeRunningUpdateCase{
				currentNodeRunningUpdate:  nodeName,
				expectedNodeRunningUpdate: nodeName,
				shouldConflict:            false,
			}),
	)
})
