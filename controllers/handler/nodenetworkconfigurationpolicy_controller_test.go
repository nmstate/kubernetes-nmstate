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

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
)

var _ = Describe("success path slot release ordering", func() {
	// buildSlotReleaseTestClient builds a reconciler + fake client where NNCP
	// status writes that release the maxUnavailable slot (count 1 -> 0) fail
	// while *failNNCPStatusWrites is true.
	buildSlotReleaseTestClient := func(failNNCPStatusWrites *bool) (
		*NodeNetworkConfigurationPolicyReconciler, client.Client, types.NamespacedName,
	) {
		nmstatectlShowFn = func() (string, error) { return "", nil }
		reconciler := &NodeNetworkConfigurationPolicyReconciler{
			RetriesUntilFail:   5,
			MaximumTimeBackoff: 30 * time.Second,
			InitialBackoff:     1 * time.Second,
		}
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkState{},
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
			&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{})
		s.AddKnownTypes(nmstatev1.GroupVersion,
			&nmstatev1.NodeNetworkConfigurationPolicy{})

		node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
		nns := nmstatev1beta1.NodeNetworkState{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
		nncp := nmstatev1.NodeNetworkConfigurationPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Status: shared.NodeNetworkConfigurationPolicyStatus{
				UnavailableNodeCountMap: map[string]int{},
			},
		}
		nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   shared.EnactmentKey(nodeName, nncp.Name).Name,
				Labels: map[string]string{shared.EnactmentPolicyLabel: nncp.Name},
			},
		}

		sawSlotClaimed := false
		clb := fake.ClientBuilder{}
		clb.WithScheme(s)
		clb.WithRuntimeObjects(&nncp, &nnce, &nns, &node)
		clb.WithStatusSubresource(&nncp, &nnce, &nns)
		clb.WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(
				ctx context.Context, cl client.Client, subResourceName string,
				obj client.Object, opts ...client.SubResourceUpdateOption,
			) error {
				if updated, isPolicy := obj.(*nmstatev1.NodeNetworkConfigurationPolicy); isPolicy {
					if updated.Status.UnavailableNodeCountMap["0"] >= 1 {
						// The increment (slot claim): let it through, remember it.
						sawSlotClaimed = true
					} else if *failNNCPStatusWrites && sawSlotClaimed {
						// The decrement (slot release, 1 -> 0): fail it.
						return apierrors.NewInternalError(context.DeadlineExceeded)
					}
				}
				return cl.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		})
		cl := clb.Build()
		reconciler.Client = cl
		reconciler.APIClient = cl
		reconciler.Log = ctrl.Log.WithName("test")
		return reconciler, cl, types.NamespacedName{Name: nncp.Name}
	}

	It("keeps the enactment Progressing when the slot release fails", func() {
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) { return "ok", nil }
		defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()
		originalSlotReleaseBackoff := slotReleaseBackoff
		slotReleaseBackoff = wait.Backoff{Duration: 1 * time.Millisecond, Steps: 1}
		defer func() { slotReleaseBackoff = originalSlotReleaseBackoff }()

		failNNCPStatusWrites := true
		reconciler, cl, policyKey := buildSlotReleaseTestClient(&failNNCPStatusWrites)

		res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(res.RequeueAfter).To(Equal(10 * time.Second))

		// The slot is still held and the enactment must NOT claim success.
		nnceKey := shared.EnactmentKey(nodeName, policyKey.Name)
		updatedNNCE := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), nnceKey, updatedNNCE)).To(Succeed())
		Expect(enactmentstatus.IsAvailable(&updatedNNCE.Status.Conditions)).To(BeFalse(),
			"enactment must not be Available while the slot is still held")
		Expect(enactmentstatus.IsProgressing(&updatedNNCE.Status.Conditions)).To(BeTrue())
	})

	// Healing convergence requires Task 4's "already holds slot" guard;
	// flip this to It when Task 4 lands.
	PIt("converges once NNCP status writes heal", func() {
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) { return "ok", nil }
		defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()

		failNNCPStatusWrites := true
		reconciler, cl, policyKey := buildSlotReleaseTestClient(&failNNCPStatusWrites)

		_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())

		// Heal: allow NNCP status writes again, reconcile converges.
		failNNCPStatusWrites = false
		_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())

		nnceKey := shared.EnactmentKey(nodeName, policyKey.Name)
		updatedNNCE := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), nnceKey, updatedNNCE)).To(Succeed())
		Expect(enactmentstatus.IsAvailable(&updatedNNCE.Status.Conditions)).To(BeTrue())

		updatedNNCP := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, updatedNNCP)).To(Succeed())
		Expect(updatedNNCP.Status.UnavailableNodeCountMap["0"]).To(Equal(0))
	})
})

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type predicateCase struct {
		GenerationOld   int64
		GenerationNew   int64
		ReconcileUpdate bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
			oldNNCP := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationOld,
				},
			}
			newNNCP := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationNew,
				},
			}

			predicate := onCreateOrUpdateWithDifferentGenerationOrDelete

			Expect(predicate.
				CreateFunc(event.TypedCreateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					Object: &newNNCP,
				})).To(BeTrue())
			Expect(predicate.
				UpdateFunc(event.TypedUpdateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					ObjectOld: &oldNNCP,
					ObjectNew: &newNNCP,
				})).To(Equal(c.ReconcileUpdate))
			Expect(predicate.
				DeleteFunc(event.TypedDeleteEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					Object: &oldNNCP,
				})).To(BeTrue())
		},
		Entry("generation remains the same",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   1,
				ReconcileUpdate: false,
			}),
		Entry("generation is different",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileUpdate: true,
			}),
	)

	type incrementUnavailableNodeCountCase struct {
		currentUnavailableNodeCount int
		expectedReconcileResult     ctrl.Result
		previousEnactmentConditions func(*shared.ConditionList, string)
	}
	DescribeTable("when claimNodeRunningUpdate is called and",
		func(c incrementUnavailableNodeCountCase) {
			nmstatectlShowFn = func() (string, error) { return "", nil }
			reconciler := NodeNetworkConfigurationPolicyReconciler{
				RetriesUntilFail:   5,
				MaximumTimeBackoff: 30 * time.Second,
				InitialBackoff:     1 * time.Second,
			}
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.GroupVersion,
				&nmstatev1beta1.NodeNetworkState{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{},
			)
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
			)

			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}

			nns := nmstatev1beta1.NodeNetworkState{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}

			nncp := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Status: shared.NodeNetworkConfigurationPolicyStatus{
					UnavailableNodeCountMap: map[string]int{},
				},
			}
			nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
				ObjectMeta: metav1.ObjectMeta{
					Name: shared.EnactmentKey(nodeName, nncp.Name).Name,
				},
				Status: shared.NodeNetworkConfigurationEnactmentStatus{},
			}

			// simulate NNCE existnce/non-existence by setting conditions
			c.previousEnactmentConditions(&nnce.Status.Conditions, "")

			objs := []runtime.Object{&nncp, &nnce, &nns, &node}

			// Create a fake client to mock API calls.
			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			clb.WithRuntimeObjects(objs...)
			clb.WithStatusSubresource(&nncp)
			clb.WithStatusSubresource(&nnce)
			clb.WithStatusSubresource(&nns)
			cl := clb.Build()

			reconciler.Client = cl
			reconciler.APIClient = cl
			reconciler.Log = ctrl.Log.WithName("controllers").WithName("NodeNetworkConfigurationPolicy")

			res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: nncp.Name},
			})

			Expect(err).To(BeNil())
			Expect(res).To(Equal(c.expectedReconcileResult))
		},

		Entry("No node applying policy with empty enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("No node applying policy with progressing enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: conditions.SetProgressing,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("No node applying policy with Pending enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: conditions.SetPending,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with empty enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with Progressing enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: conditions.SetProgressing,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with Pending enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: conditions.SetPending,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
	)

	Describe("allPolicies function", func() {
		It("should return policies in alphanumerical order by name", func() {
			// Create test policies with names in non-alphabetical order
			policies := []nmstatev1.NodeNetworkConfigurationPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "zebra-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alpha-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "beta-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "1-numeric-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "10-numeric-policy",
					},
				},
			}

			// Create a fake client with the test policies
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion, &nmstatev1.NodeNetworkConfigurationPolicy{}, &nmstatev1.NodeNetworkConfigurationPolicyList{})

			objs := make([]runtime.Object, len(policies))
			for i := range policies {
				objs[i] = &policies[i]
			}

			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			clb.WithRuntimeObjects(objs...)
			cl := clb.Build()

			// Get the allPolicies function
			allPoliciesFunc := allPolicies(cl, ctrl.Log)

			// Call the function with a dummy node
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			}

			requests := allPoliciesFunc(context.TODO(), node)

			// Verify the order is correct
			expectedOrder := []string{
				"1-numeric-policy",
				"10-numeric-policy",
				"alpha-policy",
				"beta-policy",
				"zebra-policy",
			}

			Expect(requests).To(WithTransform(func(reqs []ctrl.Request) []string {
				names := make([]string, len(reqs))
				for i, req := range reqs {
					names[i] = req.Name
				}
				return names
			}, Equal(expectedOrder)))
		})

		It("should return empty slice when no policies exist", func() {
			// Create a fake client with no policies
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion, &nmstatev1.NodeNetworkConfigurationPolicy{}, &nmstatev1.NodeNetworkConfigurationPolicyList{})

			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			cl := clb.Build()

			// Get the allPolicies function
			allPoliciesFunc := allPolicies(cl, ctrl.Log)

			// Call the function with a dummy node
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			}

			requests := allPoliciesFunc(context.TODO(), node)

			// Verify empty result
			Expect(requests).To(BeEmpty())
		})
	})

	Describe("decrementUnavailableNodeCount", func() {
		var (
			reconciler *NodeNetworkConfigurationPolicyReconciler
			nncp       *nmstatev1.NodeNetworkConfigurationPolicy
			s          *runtime.Scheme
		)

		BeforeEach(func() {
			reconciler = &NodeNetworkConfigurationPolicyReconciler{
				RetriesUntilFail:   5,
				MaximumTimeBackoff: 30 * time.Second,
				InitialBackoff:     1 * time.Second,
			}
			s = scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
				&nmstatev1.NodeNetworkConfigurationPolicyList{},
			)

			nncp = &nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Status: shared.NodeNetworkConfigurationPolicyStatus{
					UnavailableNodeCountMap: map[string]int{
						"gen-1": 2,
					},
				},
			}

			reconciler.Log = ctrl.Log.WithName("test")
		})

		Context("when status update succeeds", func() {
			It("should decrement unavailable node count and return nil", func() {
				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(nncp)
				clb.WithStatusSubresource(nncp)
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				err := reconciler.decrementUnavailableNodeCount(context.TODO(), nncp, "gen-1")

				Expect(err).To(BeNil())

				// Verify the count was decremented
				updatedNNCP := &nmstatev1.NodeNetworkConfigurationPolicy{}
				err = cl.Get(context.TODO(), types.NamespacedName{Name: "test-policy"}, updatedNNCP)
				Expect(err).To(BeNil())
				Expect(updatedNNCP.Status.UnavailableNodeCountMap["gen-1"]).To(Equal(1))
			})

			It("stamps LastUnavailableNodeCountUpdate on decrement", func() {
				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(nncp)
				clb.WithStatusSubresource(nncp)
				cl := clb.Build()
				reconciler.Client = cl
				reconciler.APIClient = cl

				Expect(reconciler.decrementUnavailableNodeCount(context.TODO(), nncp, "gen-1")).To(Succeed())

				updated := &nmstatev1.NodeNetworkConfigurationPolicy{}
				Expect(cl.Get(context.TODO(), types.NamespacedName{Name: "test-policy"}, updated)).To(Succeed())
				Expect(updated.Status.LastUnavailableNodeCountUpdate).ToNot(BeNil())
				Expect(updated.Status.LastUnavailableNodeCountUpdate.Time).To(BeTemporally("~", time.Now(), 10*time.Second))
			})
		})

		Context("when status update fails with both cached and non-cached clients", func() {
			It("should return error", func() {
				originalSlotReleaseBackoff := slotReleaseBackoff
				slotReleaseBackoff = wait.Backoff{Duration: 1 * time.Millisecond, Steps: 1}
				defer func() { slotReleaseBackoff = originalSlotReleaseBackoff }()

				// Create a client that will fail status updates
				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(nncp)
				// Note: NOT adding WithStatusSubresource - this causes status updates to fail
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				err := reconciler.decrementUnavailableNodeCount(context.TODO(), nncp, "gen-1")

				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("when unavailable node count is already zero", func() {
			It("should return nil without error", func() {
				nncpZeroCount := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-policy-zero",
					},
					Status: shared.NodeNetworkConfigurationPolicyStatus{
						UnavailableNodeCountMap: map[string]int{
							"gen-1": 0,
						},
					},
				}

				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(nncpZeroCount)
				clb.WithStatusSubresource(nncpZeroCount)
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				err := reconciler.decrementUnavailableNodeCount(context.TODO(), nncpZeroCount, "gen-1")

				// Should not return error - this is expected when node already processed
				Expect(err).To(BeNil())
			})
		})

		Context("when generation key doesn't exist", func() {
			It("should return nil without error", func() {
				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(nncp)
				clb.WithStatusSubresource(nncp)
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				// Try to decrement a generation key that doesn't exist
				err := reconciler.decrementUnavailableNodeCount(context.TODO(), nncp, "non-existent-gen")

				// Should not return error - this is expected when node already processed
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("incrementUnavailableNodeCount", func() {
		var (
			reconciler *NodeNetworkConfigurationPolicyReconciler
			nncp       *nmstatev1.NodeNetworkConfigurationPolicy
			s          *runtime.Scheme
		)

		BeforeEach(func() {
			reconciler = &NodeNetworkConfigurationPolicyReconciler{
				RetriesUntilFail:   5,
				MaximumTimeBackoff: 30 * time.Second,
				InitialBackoff:     1 * time.Second,
			}
			s = scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
				&nmstatev1.NodeNetworkConfigurationPolicyList{},
			)

			nncp = &nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy",
				},
				Status: shared.NodeNetworkConfigurationPolicyStatus{
					UnavailableNodeCountMap: map[string]int{
						"gen-1": 0,
					},
				},
			}

			reconciler.Log = ctrl.Log.WithName("test")
		})

		It("stamps LastUnavailableNodeCountUpdate on increment", func() {
			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			clb.WithRuntimeObjects(nncp)
			clb.WithStatusSubresource(nncp)
			cl := clb.Build()
			reconciler.Client = cl
			reconciler.APIClient = cl

			Expect(reconciler.incrementUnavailableNodeCount(context.TODO(), nncp, "gen-1")).To(Succeed())

			updated := &nmstatev1.NodeNetworkConfigurationPolicy{}
			Expect(cl.Get(context.TODO(), types.NamespacedName{Name: "test-policy"}, updated)).To(Succeed())
			Expect(updated.Status.LastUnavailableNodeCountUpdate).ToNot(BeNil())
		})
	})

	Describe("fillInEnactmentStatus", func() {
		var (
			reconciler *NodeNetworkConfigurationPolicyReconciler
			s          *runtime.Scheme
		)

		BeforeEach(func() {
			nmstatectlShowFn = func() (string, error) { return "", nil }
			reconciler = &NodeNetworkConfigurationPolicyReconciler{
				RetriesUntilFail:   5,
				MaximumTimeBackoff: 30 * time.Second,
				InitialBackoff:     1 * time.Second,
			}
			s = scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.GroupVersion,
				&nmstatev1beta1.NodeNetworkState{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{},
			)
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
			)
			reconciler.Log = ctrl.Log.WithName("test")
		})

		Context("when policy generation changes", func() {
			It("should reset enactment conditions", func() {
				nncp := nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test",
						Generation: 2,
					},
				}
				nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
					ObjectMeta: metav1.ObjectMeta{
						Name: shared.EnactmentKey(nodeName, nncp.Name).Name,
						Labels: map[string]string{
							shared.EnactmentPolicyLabel: nncp.Name,
						},
					},
					Status: shared.NodeNetworkConfigurationEnactmentStatus{
						PolicyGeneration: 1,
					},
				}
				// Set stale Failing conditions from previous generation
				conditions.SetFailedToConfigure(&nnce.Status.Conditions, "previous generation failure")

				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(&nncp, &nnce)
				clb.WithStatusSubresource(&nnce)
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				enactmentConditions := conditions.New(cl, shared.EnactmentKey(nodeName, nncp.Name))

				err := reconciler.fillInEnactmentStatus(context.TODO(), &nncp, &nnce, enactmentConditions)
				Expect(err).ToNot(HaveOccurred())

				updatedNNCE := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
				err = cl.Get(context.TODO(), shared.EnactmentKey(nodeName, nncp.Name), updatedNNCE)
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedNNCE.Status.PolicyGeneration).To(Equal(int64(2)))
				Expect(updatedNNCE.Status.Conditions).To(BeEmpty(),
					"conditions should be reset when generation changes")
			})
		})

		Context("when policy generation stays the same", func() {
			It("should preserve enactment conditions", func() {
				nncp := nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test",
						Generation: 1,
					},
				}
				nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
					ObjectMeta: metav1.ObjectMeta{
						Name: shared.EnactmentKey(nodeName, nncp.Name).Name,
						Labels: map[string]string{
							shared.EnactmentPolicyLabel: nncp.Name,
						},
					},
					Status: shared.NodeNetworkConfigurationEnactmentStatus{
						PolicyGeneration: 1,
					},
				}
				conditions.SetProgressing(&nnce.Status.Conditions, "Applying desired state")

				clb := fake.ClientBuilder{}
				clb.WithScheme(s)
				clb.WithRuntimeObjects(&nncp, &nnce)
				clb.WithStatusSubresource(&nnce)
				cl := clb.Build()

				reconciler.Client = cl
				reconciler.APIClient = cl

				enactmentConditions := conditions.New(cl, shared.EnactmentKey(nodeName, nncp.Name))

				err := reconciler.fillInEnactmentStatus(context.TODO(), &nncp, &nnce, enactmentConditions)
				Expect(err).ToNot(HaveOccurred())

				updatedNNCE := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
				err = cl.Get(context.TODO(), shared.EnactmentKey(nodeName, nncp.Name), updatedNNCE)
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedNNCE.Status.PolicyGeneration).To(Equal(int64(1)))
				Expect(updatedNNCE.Status.Conditions).ToNot(BeEmpty(),
					"conditions should be preserved when generation stays the same")
			})
		})
	})
})
