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
	"k8s.io/apimachinery/pkg/util/intstr"
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
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
)

// stubApplyOutput is the canned nmstate output returned by applyDesiredStateFn
// stubs in these specs.
const stubApplyOutput = "ok"

var _ = Describe("success path slot release ordering", func() {
	// buildSlotReleaseTestClient builds a reconciler + fake client where NNCP
	// status writes that release the maxUnavailable slot (count 1 -> 0) fail
	// while *failNNCPStatusWrites is true.
	buildSlotReleaseTestClient := func(failNNCPStatusWrites *bool) (
		*NodeNetworkConfigurationPolicyReconciler, client.Client, types.NamespacedName,
	) {
		originalNmstatectlShowFn := nmstatectlShowFn
		nmstatectlShowFn = func() (string, error) { return "", nil }
		DeferCleanup(func() { nmstatectlShowFn = originalNmstatectlShowFn })
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
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) { return stubApplyOutput, nil }
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

	// Note: buildSlotReleaseTestClient's interceptor couples sawSlotClaimed with
	// *failNNCPStatusWrites so only the release (decrement) write fails, never the claim.
	It("converges once NNCP status writes heal", func() {
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) { return stubApplyOutput, nil }
		defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()
		originalSlotReleaseBackoff := slotReleaseBackoff
		slotReleaseBackoff = wait.Backoff{Duration: 1 * time.Millisecond, Steps: 1}
		defer func() { slotReleaseBackoff = originalSlotReleaseBackoff }()

		failNNCPStatusWrites := true
		reconciler, cl, policyKey := buildSlotReleaseTestClient(&failNNCPStatusWrites)

		_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())

		// The enactment is now parked in the Finalizing phase; its slot is
		// released on the retry by the set-to-truth audit. In production the
		// failed decrement above burned the ~31.5s slotReleaseBackoff, so by
		// the retry the claim's LastUnavailableNodeCountUpdate is well past the
		// audit grace window; simulate that elapsed time (the spec shrinks the
		// backoff for speed) so the audit does not defer.
		parked := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), shared.EnactmentKey(nodeName, policyKey.Name), parked)).To(Succeed())
		Expect(enactmentstatus.IsFinalizing(&parked.Status.Conditions)).To(BeTrue(),
			"the first reconcile must park the enactment in the Finalizing phase")
		blocked := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, blocked)).To(Succeed())
		aged := metav1.NewTime(time.Now().Add(-2 * node.AuditGraceWindow))
		blocked.Status.LastUnavailableNodeCountUpdate = &aged
		Expect(cl.Status().Update(context.TODO(), blocked)).To(Succeed())

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

var _ = Describe("Progressing write failure after slot claim", func() {
	It("releases the just-claimed slot and requeues after 10s without applying", func() {
		originalNmstatectlShowFn := nmstatectlShowFn
		nmstatectlShowFn = func() (string, error) { return "", nil }
		defer func() { nmstatectlShowFn = originalNmstatectlShowFn }()
		applyCalled := false
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) {
			applyCalled = true
			return stubApplyOutput, nil
		}
		defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()

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
					}
				}
				if updatedNNCE, isNNCE := obj.(*nmstatev1beta1.NodeNetworkConfigurationEnactment); isNNCE {
					if sawSlotClaimed && enactmentstatus.IsProgressing(&updatedNNCE.Status.Conditions) {
						// The Progressing marker write after a successful claim: fail it.
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
		policyKey := types.NamespacedName{Name: nncp.Name}

		res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(sawSlotClaimed).To(BeTrue(), "the slot claim must have happened before the Progressing write")
		Expect(res.RequeueAfter).To(Equal(10*time.Second),
			"a failed Progressing write after a claim must requeue after 10s")
		Expect(applyCalled).To(BeFalse(),
			"apply must not run when the Progressing marker could not be persisted")

		updatedNNCP := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, updatedNNCP)).To(Succeed())
		Expect(updatedNNCP.Status.UnavailableNodeCountMap["0"]).To(Equal(0),
			"the just-claimed slot must be released when the Progressing write fails")
	})
})

var _ = Describe("success write failure after apply (finalization phase)", func() {
	// buildFinalizeTestClient builds a reconciler + fake client where NNCE
	// status writes that record success (Available=True) fail while
	// *failSuccessWrites is true. Progressing/Finalizing NNCE writes and all
	// NNCP writes are allowed.
	buildFinalizeTestClient := func(failSuccessWrites *bool, applyCalled *bool) (
		*NodeNetworkConfigurationPolicyReconciler, client.Client, types.NamespacedName,
	) {
		originalNmstatectlShowFn := nmstatectlShowFn
		nmstatectlShowFn = func() (string, error) { return "", nil }
		DeferCleanup(func() { nmstatectlShowFn = originalNmstatectlShowFn })
		originalApply := applyDesiredStateFn
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) {
			*applyCalled = true
			return stubApplyOutput, nil
		}
		DeferCleanup(func() { applyDesiredStateFn = originalApply })

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

		clb := fake.ClientBuilder{}
		clb.WithScheme(s)
		clb.WithRuntimeObjects(&nncp, &nnce, &nns, &node)
		clb.WithStatusSubresource(&nncp, &nnce, &nns)
		clb.WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(
				ctx context.Context, cl client.Client, subResourceName string,
				obj client.Object, opts ...client.SubResourceUpdateOption,
			) error {
				if nnceObj, isNNCE := obj.(*nmstatev1beta1.NodeNetworkConfigurationEnactment); isNNCE {
					if *failSuccessWrites && enactmentstatus.IsAvailable(&nnceObj.Status.Conditions) {
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

	It("requeues on success-write failure, then finalizes without re-applying", func() {
		failSuccessWrites := true
		applyCalled := false
		reconciler, cl, policyKey := buildFinalizeTestClient(&failSuccessWrites, &applyCalled)

		// Reconcile 1: applies, releases the slot, but cannot record success.
		res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(applyCalled).To(BeTrue(), "the first reconcile must apply the desired state")
		Expect(res.RequeueAfter).To(Equal(10*time.Second),
			"a failed success write must requeue rather than be swallowed")

		nnceKey := shared.EnactmentKey(nodeName, policyKey.Name)
		nnce := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), nnceKey, nnce)).To(Succeed())
		Expect(enactmentstatus.IsAvailable(&nnce.Status.Conditions)).To(BeFalse())
		Expect(enactmentstatus.IsFinalizing(&nnce.Status.Conditions)).To(BeTrue(),
			"the enactment must be parked in the finalizing phase")

		// Heal the success write and reconcile again. The retry must NOT
		// re-apply the already committed configuration; it only finalizes.
		failSuccessWrites = false
		applyCalled = false
		res, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(res).To(Equal(ctrl.Result{}))
		Expect(applyCalled).To(BeFalse(),
			"the finalization retry must not re-apply the desired state")

		Expect(cl.Get(context.TODO(), nnceKey, nnce)).To(Succeed())
		Expect(enactmentstatus.IsAvailable(&nnce.Status.Conditions)).To(BeTrue())

		nncp := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, nncp)).To(Succeed())
		Expect(nncp.Status.UnavailableNodeCountMap["0"]).To(Equal(0),
			"the slot must be released once finalization completes")
	})

	It("does not double-free another node's slot when success write fails after decrement", func() {
		// Regression for the finalization double-decrement: this node applies,
		// decrements its own slot, but fails to record success; a second node
		// still legitimately holds a slot. The finalization retry must not
		// blindly decrement again and release the other node's slot.
		originalNmstatectlShowFn := nmstatectlShowFn
		nmstatectlShowFn = func() (string, error) { return "", nil }
		defer func() { nmstatectlShowFn = originalNmstatectlShowFn }()
		applyCalled := false
		applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) {
			applyCalled = true
			return stubApplyOutput, nil
		}
		defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()

		reconciler := &NodeNetworkConfigurationPolicyReconciler{
			RetriesUntilFail: 5, MaximumTimeBackoff: 30 * time.Second, InitialBackoff: 1 * time.Second,
		}
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkState{},
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
			&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{})
		s.AddKnownTypes(nmstatev1.GroupVersion, &nmstatev1.NodeNetworkConfigurationPolicy{})

		maxUnavailable := intstr.FromInt(2)
		node1 := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
		nns := nmstatev1beta1.NodeNetworkState{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}
		oldStamp := metav1.NewTime(time.Now().Add(-2 * node.AuditGraceWindow))
		nncp := nmstatev1.NodeNetworkConfigurationPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       shared.NodeNetworkConfigurationPolicySpec{MaxUnavailable: &maxUnavailable},
			Status: shared.NodeNetworkConfigurationPolicyStatus{
				UnavailableNodeCountMap:        map[string]int{"0": 1}, // node02 already holds a slot
				LastUnavailableNodeCountUpdate: &oldStamp,
			},
		}
		nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   shared.EnactmentKey(nodeName, nncp.Name).Name,
				Labels: map[string]string{shared.EnactmentPolicyLabel: nncp.Name},
			},
		}
		// node02 is a live Progressing holder of the current generation.
		otherNNCE := nmstatev1beta1.NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   shared.EnactmentKey("node02", nncp.Name).Name,
				Labels: map[string]string{shared.EnactmentPolicyLabel: nncp.Name},
			},
			Status: shared.NodeNetworkConfigurationEnactmentStatus{PolicyGeneration: nncp.Generation},
		}
		conditions.SetProgressing(&otherNNCE.Status.Conditions, "applying")

		failSuccessWrites := true
		clb := fake.ClientBuilder{}
		clb.WithScheme(s)
		clb.WithRuntimeObjects(&nncp, &nnce, &otherNNCE, &nns, &node1)
		clb.WithStatusSubresource(&nncp, &nnce, &otherNNCE, &nns)
		clb.WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(
				ctx context.Context, cl client.Client, subResourceName string,
				obj client.Object, opts ...client.SubResourceUpdateOption,
			) error {
				if nnceObj, isNNCE := obj.(*nmstatev1beta1.NodeNetworkConfigurationEnactment); isNNCE {
					if failSuccessWrites && nnceObj.Name == nnce.Name &&
						enactmentstatus.IsAvailable(&nnceObj.Status.Conditions) {
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
		policyKey := types.NamespacedName{Name: nncp.Name}

		// Reconcile 1: claim (1->2), apply, decrement (2->1), success write fails.
		res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(applyCalled).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(10 * time.Second))
		afterFirst := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, afterFirst)).To(Succeed())
		Expect(afterFirst.Status.UnavailableNodeCountMap["0"]).To(Equal(1),
			"this node's slot is released; node02's slot remains")

		// Reconcile 2: heal, finalize. Must NOT decrement node02's slot away.
		failSuccessWrites = false
		applyCalled = false
		_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: policyKey})
		Expect(err).To(BeNil())
		Expect(applyCalled).To(BeFalse(), "finalization retry must not re-apply")

		finalNNCP := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, finalNNCP)).To(Succeed())
		Expect(finalNNCP.Status.UnavailableNodeCountMap["0"]).To(Equal(1),
			"node02's slot must be preserved; a blind re-decrement would break maxUnavailable")
		updatedNNCE := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), shared.EnactmentKey(nodeName, nncp.Name), updatedNNCE)).To(Succeed())
		Expect(enactmentstatus.IsAvailable(&updatedNNCE.Status.Conditions)).To(BeTrue())
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
		lastCountUpdateAge          time.Duration // 0 = nil timestamp
		otherNodeLiveHolder         bool          // add a fresh Progressing enactment owned by another node
		previousEnactmentConditions func(*shared.ConditionList, string)
		expectBlocked               bool
		expectedReconcileResult     ctrl.Result // only checked when !expectBlocked
	}
	DescribeTable("when claimNodeRunningUpdate is called and",
		func(c incrementUnavailableNodeCountCase) {
			originalNmstatectlShowFn := nmstatectlShowFn
			nmstatectlShowFn = func() (string, error) { return "", nil }
			defer func() { nmstatectlShowFn = originalNmstatectlShowFn }()
			// "Proceeds" entries must complete the claim + apply path
			// deterministically, so stub the apply to succeed; blocked entries
			// never reach the apply, the stub is irrelevant there.
			applyDesiredStateFn = func(context.Context, client.Client, shared.State) (string, error) { return stubApplyOutput, nil }
			defer func() { applyDesiredStateFn = nmstate.ApplyDesiredState }()
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
					UnavailableNodeCountMap: map[string]int{
						"0": c.currentUnavailableNodeCount, // policy generation is 0
					},
				},
			}
			if c.lastCountUpdateAge > 0 {
				nncp.Status.LastUnavailableNodeCountUpdate = &metav1.Time{Time: time.Now().Add(-c.lastCountUpdateAge)}
			}
			nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
				ObjectMeta: metav1.ObjectMeta{
					Name:   shared.EnactmentKey(nodeName, nncp.Name).Name,
					Labels: map[string]string{shared.EnactmentPolicyLabel: nncp.Name},
				},
				Status: shared.NodeNetworkConfigurationEnactmentStatus{},
			}

			// simulate NNCE existnce/non-existence by setting conditions
			c.previousEnactmentConditions(&nnce.Status.Conditions, "")

			objs := []runtime.Object{&nncp, &nnce, &nns, &node}
			if c.otherNodeLiveHolder {
				otherNNCE := nmstatev1beta1.NodeNetworkConfigurationEnactment{
					ObjectMeta: metav1.ObjectMeta{
						Name:   shared.EnactmentKey("node02", nncp.Name).Name,
						Labels: map[string]string{shared.EnactmentPolicyLabel: nncp.Name},
					},
					Status: shared.NodeNetworkConfigurationEnactmentStatus{
						PolicyGeneration: nncp.Generation,
					},
				}
				// SetProgressing writes the full condition set (shouldAbortReconcile
				// requires it) with a fresh heartbeat, making node02 a live holder.
				conditions.SetProgressing(&otherNNCE.Status.Conditions, "applying")
				objs = append(objs, &otherNNCE)
			}

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
			if c.expectBlocked {
				Expect(res.RequeueAfter).To(BeNumerically(">=", 90*time.Second))
				Expect(res.RequeueAfter).To(BeNumerically("<", 120*time.Second))
			} else {
				Expect(res).To(Equal(c.expectedReconcileResult))
			}
		},

		// "proceeds" entries expect ctrl.Result{}: with applyDesiredStateFn
		// stubbed to succeed, a successful claim runs apply + slot release +
		// NotifySuccess and the reconcile completes without requeueing.
		Entry("count 0, empty enactment -> claims slot and proceeds",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectBlocked:               false,
				expectedReconcileResult:     ctrl.Result{},
			}),
		Entry("count at cap with fresh live holder on another node -> blocked with bounded requeue",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				lastCountUpdateAge:          5 * time.Minute,
				otherNodeLiveHolder:         true,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectBlocked:               true,
			}),
		Entry("count at cap, no live holder, stale timestamp -> audit repairs and proceeds",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				lastCountUpdateAge:          5 * time.Minute,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectBlocked:               false,
				expectedReconcileResult:     ctrl.Result{},
			}),
		Entry("count at cap, no live holder, fresh timestamp -> grace defers, blocked",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				lastCountUpdateAge:          5 * time.Second,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectBlocked:               true,
			}),
		Entry("own enactment Progressing (interrupted apply) -> already holds slot, proceeds without increment",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				lastCountUpdateAge:          5 * time.Second, // grace would block; guard must bypass
				previousEnactmentConditions: conditions.SetProgressing,
				expectBlocked:               false,
				expectedReconcileResult:     ctrl.Result{},
			}),
		Entry("own enactment Pending at cap with live holder -> stays blocked",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				lastCountUpdateAge:          5 * time.Minute,
				otherNodeLiveHolder:         true,
				previousEnactmentConditions: conditions.SetPending,
				expectBlocked:               true,
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
			originalNmstatectlShowFn := nmstatectlShowFn
			nmstatectlShowFn = func() (string, error) { return "", nil }
			DeferCleanup(func() { nmstatectlShowFn = originalNmstatectlShowFn })
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
