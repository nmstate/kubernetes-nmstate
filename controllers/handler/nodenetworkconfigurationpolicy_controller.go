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
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/bridge"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmpolicy"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/policyconditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/selectors"
)

const (
	ReconcileFailed = "ReconcileFailed"
)

// blockedRequeueBase/Jitter bound recovery when a policy is throttled: the
// reconcile re-checks within [90s, 120s) even on a quiet cluster instead of
// waiting for watch events or the multi-hour cache resync.
const (
	blockedRequeueBase   = 90 * time.Second
	blockedRequeueJitter = 30 * time.Second
)

func blockedRequeueResult() ctrl.Result {
	//nolint:gosec // jitter is not security-sensitive, math/rand is fine
	return ctrl.Result{RequeueAfter: blockedRequeueBase + time.Duration(rand.Int63n(int64(blockedRequeueJitter)))}
}

var (
	nodeName                                        string
	onCreateOrUpdateWithDifferentGenerationOrDelete = predicate.TypedFuncs[*nmstatev1.NodeNetworkConfigurationPolicy]{
		CreateFunc: func(createEvent event.TypedCreateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.TypedDeleteEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.TypedUpdateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) bool {
			// [1] https://blog.openshift.com/kubernetes-operators-best-practices/
			generationIsDifferent := updateEvent.ObjectNew.GetGeneration() != updateEvent.ObjectOld.GetGeneration()
			return generationIsDifferent
		},
	}

	onLabelsUpdatedForThisNode = predicate.TypedFuncs[*corev1.Node]{
		CreateFunc: func(createEvent event.TypedCreateEvent[*corev1.Node]) bool {
			return false
		},
		DeleteFunc: func(event.TypedDeleteEvent[*corev1.Node]) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.TypedUpdateEvent[*corev1.Node]) bool {
			labelsChanged := !reflect.DeepEqual(updateEvent.ObjectOld.GetLabels(), updateEvent.ObjectNew.GetLabels())
			return labelsChanged && node.EventIsForThisNode(updateEvent.ObjectNew)
		},
		GenericFunc: func(event.TypedGenericEvent[*corev1.Node]) bool {
			return false
		},
	}
	nmstatectlShowFn    = nmstatectl.Show
	applyDesiredStateFn = nmstate.ApplyDesiredState
)

// NodeNetworkConfigurationPolicyReconciler reconciles a NodeNetworkConfigurationPolicy object
type NodeNetworkConfigurationPolicyReconciler struct {
	client.Client
	// APIClient controller-runtime client without cache, it will be used at
	// places where whole cluster resources need to be retrieved but not cached.
	APIClient client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	// RetriesUntilFail is the number of retry attempts before marking an NNCE as failed.
	// Expected range: >= 1. Defaults to 5 via NNCP_MAX_RETRIES env var.
	RetriesUntilFail int
	// MaximumTimeBackoff is the upper bound for exponential backoff between retries.
	// Expected range: > 0. Defaults to 30s via NNCP_MAX_BACKOFF_SECONDS env var.
	MaximumTimeBackoff time.Duration
	// InitialBackoff is the starting backoff duration for the first retry.
	// Expected range: > 0. Defaults to 1s via NNCP_INITIAL_BACKOFF_SECONDS env var.
	InitialBackoff time.Duration
}

func init() {
	if !environment.IsHandler() {
		return
	}

	nodeName = environment.NodeName()
	if nodeName == "" {
		panic("NODE_NAME is mandatory")
	}
}

// Reconcile reads the state of the cluster for a NodeNetworkConfigurationPolicy object and makes changes based on the state read
// and what is in the NodeNetworkConfigurationPolicy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
//
//nolint:funlen,gocyclo
func (r *NodeNetworkConfigurationPolicyReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nodenetworkconfigurationpolicy", request.NamespacedName)

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1.NodeNetworkConfigurationPolicy{}
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Policy is not found, removing previous enactment if any")
			err = r.deleteEnactmentForPolicy(ctx, request.Name)
			return ctrl.Result{}, err
		}
		log.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if !policyconditions.IsProgressing(&instance.Status.Conditions) {
		policyconditions.Reset(ctx, r.Client, request.NamespacedName)
	}

	// Policy conditions will be updated at the end so updating it
	// does not impact at applying state, it will increase just
	// reconcile time.
	defer policyconditions.Update(ctx, r.Client, r.APIClient, request.NamespacedName)

	policySelectors := selectors.NewFromPolicy(r.Client, instance)
	unmatchingNodeLabels, err := policySelectors.UnmatchedNodeLabels(ctx, nodeName)
	if err != nil {
		log.Error(err, "failed checking node selectors")
		return ctrl.Result{}, err
	}

	if len(unmatchingNodeLabels) > 0 {
		log.Info("Policy node selectors does not match node, removing previous enactment if any")
		err = r.deleteEnactmentForPolicy(ctx, request.Name)
		return ctrl.Result{}, err
	}

	enactmentInstance, err := r.initializeEnactment(ctx, instance)
	if err != nil {
		log.Error(err, "Error initializing enactment")
		return ctrl.Result{}, err
	}
	previousConditions := &enactmentInstance.Status.Conditions
	enactmentConditions := enactmentconditions.New(r.APIClient, nmstateapi.EnactmentKey(nodeName, instance.Name))

	err = r.fillInEnactmentStatus(ctx, instance, enactmentInstance, enactmentConditions)
	if err != nil {
		log.Error(err, "failed filling in the NNCE status")
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	enactmentInstance, err = r.enactmentForPolicy(ctx, instance)
	if err != nil {
		log.Error(err, "error getting enactment for policy")
		return ctrl.Result{}, err
	}

	generationKey := strconv.FormatInt(enactmentInstance.Status.PolicyGeneration, 10)

	// Verify the policy still exists via uncached client before applying.
	// The cached client may return stale data if the informer watch was
	// broken during a previous network-disrupting apply cycle.
	freshPolicy := &nmstatev1.NodeNetworkConfigurationPolicy{}
	if err := r.APIClient.Get(ctx, request.NamespacedName, freshPolicy); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Policy no longer exists (verified via API server), removing enactment")
			err = r.deleteEnactmentForPolicy(ctx, request.Name)
			return ctrl.Result{}, err
		}
		log.Error(err, "Failed to verify policy existence via API server")
		return ctrl.Result{}, err
	}

	// Skip apply if retries are already exhausted for this generation.
	// This prevents unnecessary network disruption when a spurious reconcile
	// (e.g., from informer re-list after reconnection) re-triggers processing
	// of an already-failed policy.
	if enactmentInstance.Status.RetryCount[generationKey] >= r.RetriesUntilFail {
		log.Info("Retry count already exhausted, skipping apply",
			"retryCount", enactmentInstance.Status.RetryCount[generationKey],
			"maxRetries", r.RetriesUntilFail,
			"generation", generationKey)
		return ctrl.Result{}, nil
	}

	// Already fully reconciled for this generation: a spurious re-reconcile
	// (informer re-list, node label change) must not re-claim a slot and
	// re-apply an already committed configuration. Release the slot with the
	// idempotent audit in case an earlier finalize recorded success but could
	// not release it, then finish.
	if enactmentstatus.IsAvailable(&enactmentInstance.Status.Conditions) {
		log.Info("enactment already Available for current generation, ensuring slot released")
		if err := r.releaseUnavailableSlotByAudit(ctx, request.NamespacedName); err != nil {
			log.Info("Available enactment: unavailable-node slot release will be retried",
				"error", err, "requeueAfter", "10s")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	// Post-apply finalization phase: the desired state was applied in an
	// earlier reconcile but the slot release or success write did not complete.
	// Only finalize (record success, release the slot); do NOT re-apply the
	// already committed configuration.
	if enactmentstatus.IsFinalizing(&enactmentInstance.Status.Conditions) {
		log.Info("enactment already applied (finalizing); recording success and releasing slot without re-applying")
		return r.finalizeInterruptedApply(ctx, request.NamespacedName, enactmentConditions), nil
	}

	alreadyHoldsSlot := enactmentstatus.IsProgressing(&enactmentInstance.Status.Conditions)
	if alreadyHoldsSlot {
		log.Info("enactment already Progressing for current generation; slot held by an interrupted reconcile, skipping claim")
	}
	didClaim := false
	if !alreadyHoldsSlot && r.shouldIncrementUnavailableNodeCount(previousConditions) {
		err = r.claimUnavailableSlot(ctx, instance, request.NamespacedName, generationKey)
		if err != nil {
			if apierrors.IsConflict(err) || errors.Is(err, node.MaxUnavailableLimitReachedError{}) {
				enactmentConditions.NotifyPending(ctx)
				log.Info(err.Error())
				shouldAbortEnactment, err := r.shouldAbortReconcile(ctx, instance)
				if err != nil {
					return ctrl.Result{}, err
				}
				if shouldAbortEnactment {
					if r.Recorder != nil {
						r.Recorder.Event(
							instance,
							corev1.EventTypeWarning,
							ReconcileFailed,
							fmt.Errorf("reconciliation of enactment %q has aborted", enactmentInstance.Name).Error())
					}
					enactmentConditions.NotifyAborted(ctx, fmt.Errorf("reconciliation of enactment %q has aborted", enactmentInstance.Name))
					return ctrl.Result{}, nil
				}
				return blockedRequeueResult(), nil
			}
			return ctrl.Result{}, err
		}
		didClaim = true
	}

	if err := enactmentConditions.NotifyProgressing(ctx); err != nil {
		// Without a persisted Progressing marker the audit on other nodes
		// cannot see this node as a live slot holder. Do not apply: release
		// the slot claimed in this reconcile (if any) and retry shortly.
		//
		// Use a bounded release that stays within node.AuditGraceWindow so it
		// cannot race the audit: a longer retry could land after the grace
		// expired and another node had already audited this markerless claim
		// away and taken the slot, double-freeing it. If the bounded release
		// fails, the set-to-truth audit reclaims the slot (this enactment is
		// not Progressing, so it never counts as a live holder).
		if didClaim {
			if releaseErr := tryDecrementingUnavailableNodeCount(
				ctx, r.Client, r.APIClient, request.NamespacedName, generationKey, compensatingReleaseBackoff,
			); releaseErr != nil {
				log.Error(releaseErr,
					"failed releasing just-claimed slot after Progressing write failure; audit will reclaim it")
			}
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if policyconditions.IsUnknown(&instance.Status.Conditions) {
		policyconditions.Update(ctx, r.Client, r.APIClient, request.NamespacedName)
	}

	nmstateOutput, err := applyDesiredStateFn(ctx, r.APIClient, enactmentInstance.Status.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy on node %s at desired state apply: %q,\n %v",
			nodeName, nmstateOutput, err)
		log.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))
		err := r.incrementNNCERetryCount(ctx, instance, enactmentInstance, generationKey)
		if err != nil {
			log.Info("Error incrementing NNCERetry count")
			return ctrl.Result{}, err
		}

		if enactmentInstance.Status.RetryCount[generationKey] >= r.RetriesUntilFail {
			enactmentConditions.NotifyFailedToConfigure(ctx, errmsg)
			if r.Recorder != nil {
				r.Recorder.Event(instance,
					corev1.EventTypeWarning,
					ReconcileFailed,
					fmt.Errorf(
						"reconciliation of enactment %q has failed after %d retries",
						enactmentInstance.Name, r.RetriesUntilFail).Error())
			}
			return ctrl.Result{}, nil
		}
		enactmentConditions.NotifyRetrying(
			ctx,
			fmt.Errorf("failed to reconcile NodeNetworkConfigurationPolicy on node %s. Retrying %d/%d",
				nodeName,
				enactmentInstance.Status.RetryCount[generationKey]+1,
				r.RetriesUntilFail),
		)
		return ctrl.Result{Requeue: true}, nil
	}
	log.Info("nmstate", "output", nmstateOutput)

	// Enter the finalization phase before touching the counter. The enactment
	// stays Progressing (a live slot holder) but is marked Finalizing, so if
	// the release or success write below fails, the requeue finalizes at the
	// top of Reconcile instead of re-applying the already committed
	// configuration. The marker write itself gets the authoritative retry
	// budget because it runs right after the node reconfigured its own
	// networking, when the API server may be briefly unreachable; only if it
	// cannot be persisted at all does the reconcile fall back to a (safe,
	// idempotent) re-apply on the next pass.
	if err := retry.OnError(slotReleaseBackoff, func(error) bool { return true }, func() error {
		return enactmentConditions.NotifyFinalizing(ctx)
	}); err != nil {
		r.Log.Info("Failed to record finalizing phase, will retry",
			"error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Fast-path release: a blind decrement is safe here because the enactment
	// is still Progressing (Finalizing reports Progressing=True), so a
	// concurrent audit on another node counts it as a live holder and will not
	// free its slot. Success is recorded only after the slot is released, so
	// the poisonous Available+held-slot state stays unreachable.
	if err := r.decrementUnavailableNodeCount(ctx, instance, generationKey); err != nil {
		r.Log.Info("Failed to release unavailable-node slot, will retry",
			"error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if err := enactmentConditions.NotifySuccess(ctx); err != nil {
		// The slot is released, but success was not persisted. Do not swallow
		// this: the enactment would stay Finalizing and, because this
		// controller watches neither NNCE updates nor status-only NNCP
		// updates, nothing would re-trigger reconciliation and the policy
		// would stay Progressing forever. Requeue so a later reconcile
		// finalizes (records success) without re-applying.
		r.Log.Info("Failed to record enactment success, will retry",
			"error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	r.forceNNSRefresh(ctx, nodeName)

	return ctrl.Result{}, nil
}

// finalizeInterruptedApply completes a reconcile whose apply already committed
// (the enactment is in the Finalizing phase) but whose slot release or success
// write did not finish. It never re-applies the desired state.
//
// Unlike the happy path, the release here must be idempotent: a previous
// reconcile may already have decremented the slot before failing to record
// success, and a second blind decrement could drop another node's slot and
// break maxUnavailable. Success is therefore recorded first (moving this
// enactment out of the live-holder set), and the slot is then released with the
// set-to-truth audit, which repairs the counter to the number of live holders
// and can never free a slot another node still holds.
func (r *NodeNetworkConfigurationPolicyReconciler) finalizeInterruptedApply(
	ctx context.Context,
	policyKey types.NamespacedName,
	enactmentConditions enactmentconditions.EnactmentConditions,
) ctrl.Result {
	if err := enactmentConditions.NotifySuccess(ctx); err != nil {
		r.Log.Info("Failed to record enactment success, will retry",
			"error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}
	}
	if err := r.releaseUnavailableSlotByAudit(ctx, policyKey); err != nil {
		r.Log.Info("Failed to release unavailable-node slot, will retry",
			"error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}
	}
	r.forceNNSRefresh(ctx, nodeName)
	return ctrl.Result{}
}

// releaseUnavailableSlotByAudit releases a slot idempotently by repairing the
// policy counter down to the number of live Progressing holders. The caller
// must have already moved this enactment out of the live-holder set (it is
// Available), so the audit drops this node's slot. Being set-to-truth it is
// idempotent and never frees a slot another node still holds, so it is safe to
// run repeatedly and concurrently with audits on other nodes. It uses the
// larger slotReleaseBackoff because it runs right after the node reconfigured
// its own networking, when the API server may be briefly unreachable.
func (r *NodeNetworkConfigurationPolicyReconciler) releaseUnavailableSlotByAudit(
	ctx context.Context,
	policyKey types.NamespacedName,
) error {
	return retry.OnError(slotReleaseBackoff, func(error) bool { return true }, func() error {
		_, err := node.AuditUnavailableSlots(ctx, r.Client, r.APIClient, policyKey, node.StaleEnactmentThreshold())
		return err
	})
}

func (r *NodeNetworkConfigurationPolicyReconciler) incrementNNCERetryCount(
	ctx context.Context,
	instance *nmstatev1.NodeNetworkConfigurationPolicy,
	enactment *nmstatev1beta1.NodeNetworkConfigurationEnactment,
	generationKey string) error {
	if enactment.Status.RetryCount == nil {
		enactment.Status.RetryCount = map[string]int{}
	}
	count := enactment.Status.RetryCount[generationKey]

	enactment.Status.RetryCount[generationKey] = count + 1
	return enactmentstatus.Update(
		ctx,
		r.APIClient,
		nmstateapi.EnactmentKey(nodeName, instance.Name),
		func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
			status.RetryCount = enactment.Status.RetryCount
		},
	)
}

func (r *NodeNetworkConfigurationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	allPoliciesFunc := allPolicies(r.Client, r.Log)

	// Reconcile NNCP if they are created/updated/deleted or
	// Node is updated (for example labels are changed), node creation event
	// is not needed since all NNCPs are going to be Reconcile at node startup.
	c, err := controller.New(
		"NodeNetworkConfigurationPolicy",
		mgr,
		controller.Options{
			Reconciler:  r,
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](r.InitialBackoff, r.MaximumTimeBackoff),
		})
	if err != nil {
		return errors.Wrap(err, "failed to create NodeNetworkConfigurationPolicy controller")
	}

	// Add watch for NNCP
	err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&nmstatev1.NodeNetworkConfigurationPolicy{},
			&handler.TypedEnqueueRequestForObject[*nmstatev1.NodeNetworkConfigurationPolicy]{},
			onCreateOrUpdateWithDifferentGenerationOrDelete,
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to add watch for NNCPs")
	}

	// Add watch to enque all NNCPs on nod label changes
	err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&corev1.Node{},
			handler.TypedEnqueueRequestsFromMapFunc[*corev1.Node](allPoliciesFunc),
			onLabelsUpdatedForThisNode,
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to add watch to enqueue NNCPs reconcile on node label change")
	}

	return nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) initializeEnactment(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
) (*nmstatev1beta1.NodeNetworkConfigurationEnactment, error) {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	log := r.Log.WithName("initializeEnactment").WithValues("policy", policy.Name, "enactment", enactmentKey.Name)
	// Return if it's already initialize or we cannot retrieve it
	enactmentInstance := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.APIClient.Get(ctx, enactmentKey, &enactmentInstance)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed getting enactment ")
	}
	if err != nil && apierrors.IsNotFound(err) {
		// Re-fetch policy from API server and re-check selector before creating enactment
		// to prevent race condition where cached policy data might be stale
		freshPolicy := &nmstatev1.NodeNetworkConfigurationPolicy{}
		if err := r.APIClient.Get(ctx, types.NamespacedName{Name: policy.Name}, freshPolicy); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Policy no longer exists, skipping enactment creation")
				return nil, errors.Wrapf(err, "enactment policy %v does not exist", policy.Name)
			}
			return nil, errors.Wrap(err, "failed re-fetching policy from API server")
		}

		// Re-check node selector with fresh policy data
		policySelectors := selectors.NewFromPolicy(r.APIClient, freshPolicy)
		unmatchingLabels, err := policySelectors.UnmatchedNodeLabels(ctx, nodeName)
		if err != nil {
			return nil, errors.Wrap(err, "failed re-checking node selectors")
		}
		if len(unmatchingLabels) > 0 {
			return nil, fmt.Errorf(
				"node selector no longer matches after re-check, skipping enactment creation, non-matching labels: %v",
				unmatchingLabels)
		}

		log.Info("creating enactment")
		// Fetch the Node instance
		nodeInstance := &corev1.Node{}
		err = r.APIClient.Get(ctx, types.NamespacedName{Name: nodeName}, nodeInstance)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting node")
		}
		enactmentInstance = nmstatev1beta1.NewEnactment(nodeInstance, policy)
		err = r.APIClient.Create(ctx, &enactmentInstance)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating NodeNetworkConfigurationEnactment: %+v", enactmentInstance)
		}
		err = r.waitEnactmentCreated(ctx, enactmentKey)
		if err != nil {
			return nil, errors.Wrapf(err, "error waitting for NodeNetworkConfigurationEnactment: %+v", enactmentInstance)
		}
		if err := enactmentstatus.Update(ctx, r.APIClient, enactmentKey, func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
			*status = enactmentInstance.Status
		}); err != nil {
			return nil, errors.Wrapf(err, "error updating NodeNetworkConfigurationEnactment.Status on creation: %+v", enactmentInstance)
		}
		// Refresh nnce instance
		err = r.APIClient.Get(ctx, enactmentKey, &enactmentInstance)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting created enactment after updating status: %+v", enactmentInstance)
		}
	}
	return &enactmentInstance, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) fillInEnactmentStatus(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	enactmentInstance *nmstatev1beta1.NodeNetworkConfigurationEnactment,
	enactmentConditions enactmentconditions.EnactmentConditions) error {
	log := r.Log.WithValues("nodenetworkconfigurationpolicy.fillInEnactmentStatus", enactmentInstance.Name)
	currentState, err := nmstatectlShowFn()
	if err != nil {
		return err
	}

	capturedStates, generatedDesiredState, err := nmpolicy.GenerateState(
		policy.Spec.DesiredState,
		policy.Spec,
		nmstateapi.NewState(currentState),
		enactmentInstance.Status.CapturedStates,
	)
	if err != nil {
		err2 := enactmentstatus.Update(
			ctx,
			r.APIClient,
			nmstateapi.EnactmentKey(nodeName, policy.Name),
			func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
				resetPolicyGeneration(status, policy.Generation)
			},
		)
		if err2 != nil {
			return err2
		}
		enactmentConditions.NotifyGenerateFailure(ctx, err)
		return err
	}

	desiredStateWithDefaults, err := bridge.ApplyDefaultVlanFiltering(generatedDesiredState)
	if err != nil {
		return err
	}

	features := []string{}
	stats, err := nmstatectl.Statistic(desiredStateWithDefaults)
	if err != nil {
		log.Error(err, "failed calculating nmstate statistics")
	} else {
		for feature := range stats.Features {
			features = append(features, feature)
		}
	}

	return enactmentstatus.Update(
		ctx,
		r.APIClient,
		nmstateapi.EnactmentKey(nodeName, policy.Name),
		func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
			resetPolicyGeneration(status, policy.Generation)
			status.DesiredState = desiredStateWithDefaults
			status.CapturedStates = capturedStates
			status.Features = features
		},
	)
}

// resetPolicyGeneration updates the enactment's PolicyGeneration and clears
// stale conditions when the generation changes. This prevents
// policyconditions.Update on other handlers from misattributing
// previous-generation failure conditions to the new generation.
func resetPolicyGeneration(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus, generation int64) {
	if status.PolicyGeneration != generation {
		status.Conditions = nmstateapi.ConditionList{}
	}
	status.PolicyGeneration = generation
}

func (r *NodeNetworkConfigurationPolicyReconciler) enactmentForPolicy(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
) (*nmstatev1beta1.NodeNetworkConfigurationEnactment, error) {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	instance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.APIClient.Get(ctx, enactmentKey, instance)
	if err != nil {
		return nil, errors.Wrap(err, "getting enactment failed")
	}
	return instance, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) waitEnactmentCreated(ctx context.Context, enactmentKey types.NamespacedName) error {
	var enactmentInstance nmstatev1beta1.NodeNetworkConfigurationEnactment
	interval := time.Second
	timeout := 10 * time.Second
	pollErr := wait.PollUntilContextTimeout(ctx, interval, timeout, true, /*immediate*/
		func(ctx context.Context) (bool, error) {
			err := r.APIClient.Get(ctx, enactmentKey, &enactmentInstance)
			if err != nil {
				if apierrors.IsNotFound(err) {
					// Let's retry after a while, sometimes it takes some time
					// for enactment to be created
					return false, nil
				}
				return false, err
			}
			return true, nil
		})

	return pollErr
}

func (r *NodeNetworkConfigurationPolicyReconciler) deleteEnactmentForPolicy(ctx context.Context, policyName string) error {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policyName)
	log := r.Log.WithName("deleteEnactmentForPolicy").WithValues(
		"policy", policyName,
		"enactment", enactmentKey.Name,
	)
	enactmentInstance := nmstatev1beta1.NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name: enactmentKey.Name,
		},
	}
	err := r.APIClient.Delete(ctx, &enactmentInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("no enactment to delete")
			return nil
		}
		return errors.Wrap(err, "failed deleting enactment")
	}
	return nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) shouldIncrementUnavailableNodeCount(
	conditions *nmstateapi.ConditionList) bool {
	log := r.Log.WithName("shouldIncrementUnavailableNodeCount").WithValues(
		"conditions", conditions)
	shouldIncrement := conditions != nil && !enactmentstatus.IsRetrying(conditions)
	log.Info("shouldIncrementUnavailableNodeCount", "shouldIncrement", shouldIncrement)
	return shouldIncrement
}

// claimUnavailableSlot increments the policy's unavailable-node counter. If
// the counter is at the maxUnavailable cap, it audits the counter against
// live Progressing enactments (repairing ghost slots left by interrupted
// applies) and retries the increment once.
func (r *NodeNetworkConfigurationPolicyReconciler) claimUnavailableSlot(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	policyKey types.NamespacedName,
	generationKey string,
) error {
	err := r.incrementUnavailableNodeCount(ctx, policy, generationKey)
	if err == nil || !errors.Is(err, node.MaxUnavailableLimitReachedError{}) {
		return err
	}
	repaired, auditErr := node.AuditUnavailableSlots(
		ctx, r.Client, r.APIClient, policyKey, node.StaleEnactmentThreshold())
	if auditErr != nil {
		r.Log.Error(auditErr, "unavailable-slot audit failed", "policy", policyKey.Name)
		return err
	}
	if !repaired {
		return err
	}
	r.Log.Info("unavailable-slot audit repaired ghost slots, retrying claim", "policy", policyKey.Name)
	return r.incrementUnavailableNodeCount(ctx, policy, generationKey)
}

func (r *NodeNetworkConfigurationPolicyReconciler) incrementUnavailableNodeCount(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	generationKey string) error {
	policyKey := types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}
	return retry.OnError(retry.DefaultRetry, func(error) bool { return true }, func() error {
		err := r.Get(ctx, policyKey, policy)
		if err != nil {
			return err
		}
		maxUnavailable, err := node.MaxUnavailableNodeCount(ctx, r.APIClient, policy)
		if err != nil {
			r.Log.Info(
				fmt.Sprintf("failed calculating limit of max unavailable nodes, defaulting to %d, err: %s", maxUnavailable, err.Error()),
			)
		}

		if policy.Status.UnavailableNodeCountMap == nil {
			policy.Status.UnavailableNodeCountMap = map[string]int{}
		}
		if policy.Status.UnavailableNodeCountMap[generationKey] >= maxUnavailable {
			return node.MaxUnavailableLimitReachedError{}
		}
		policy.Status.UnavailableNodeCountMap[generationKey] += 1
		now := metav1.Now()
		policy.Status.LastUnavailableNodeCountUpdate = &now
		return r.Client.Status().Update(ctx, policy)
	})
}

// slotReleaseBackoff is the retry budget for the authoritative (non-cached)
// unavailable-slot release attempt. It runs right after the node's own
// networking was reconfigured, so it deserves a much larger budget than the
// cached fast-path.
//
// wait.Backoff.Steps counts attempts, not sleeps: ExponentialBackoff sleeps
// between attempts but not after the last one, so N steps yield N-1 sleeps.
// Seven steps therefore sleep 0.5+1+2+4+8+16 = ~31.5s cumulative across six
// waits. The enactment is still Progressing (a live slot holder) throughout
// this release, so a concurrent audit on another node counts it as live and
// will not free its slot, keeping the release safe even past the audit grace.
var slotReleaseBackoff = wait.Backoff{
	Duration: 500 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0.1,
	Steps:    7, // 6 sleeps: 0.5+1+2+4+8+16 = ~31.5s cumulative
}

// compensatingReleaseBackoff bounds the best-effort release of a slot that was
// claimed in this reconcile but whose Progressing marker could not be
// persisted. Unlike the success-path release, this one MUST finish within
// node.AuditGraceWindow: the claim just stamped LastUnavailableNodeCountUpdate,
// so audits on other nodes defer for that window. Releasing inside it
// guarantees no other node can audit the (markerless) claim away and take the
// slot before this release lands, which would turn a late decrement into a
// double-free. If it still fails, the enactment is not Progressing, so the
// set-to-truth audit will reclaim the slot safely.
var compensatingReleaseBackoff = wait.Backoff{
	Duration: 500 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0.1,
	Steps:    5, // 4 sleeps: 0.5+1+2+4 = ~7.5s cumulative, well under AuditGraceWindow (30s)
}

func (r *NodeNetworkConfigurationPolicyReconciler) decrementUnavailableNodeCount(
	ctx context.Context,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	generationKey string) error {
	policyKey := types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}
	err := tryDecrementingUnavailableNodeCount(ctx, r.Client, r.Client, policyKey, generationKey, retry.DefaultRetry)
	if err != nil {
		r.Log.Error(err, "error decrementing unavailableNodeCount with cached client, retrying with non-cached client and larger budget.")
		err = tryDecrementingUnavailableNodeCount(ctx, r.Client, r.APIClient, policyKey, generationKey, slotReleaseBackoff)
		if err != nil {
			r.Log.Error(err, "error decrementing unavailableNodeCount with non-cached client")
			return err
		}
	}
	return nil
}

func tryDecrementingUnavailableNodeCount(
	ctx context.Context,
	statusWriterClient client.StatusClient,
	readerClient client.Reader,
	policyKey types.NamespacedName,
	generationKey string,
	backoff wait.Backoff) error {
	instance := &nmstatev1.NodeNetworkConfigurationPolicy{}
	err := retry.OnError(backoff, func(error) bool { return true }, func() error {
		err := readerClient.Get(ctx, policyKey, instance)
		if err != nil {
			return err
		}
		if instance.Status.UnavailableNodeCountMap == nil {
			instance.Status.UnavailableNodeCountMap = map[string]int{}
		}
		if instance.Status.UnavailableNodeCountMap[generationKey] <= 0 {
			return nil
		}
		instance.Status.UnavailableNodeCountMap[generationKey] -= 1
		now := metav1.Now()
		instance.Status.LastUnavailableNodeCountUpdate = &now
		return statusWriterClient.Status().Update(ctx, instance)
	})
	return err
}

func (r *NodeNetworkConfigurationPolicyReconciler) forceNNSRefresh(ctx context.Context, name string) {
	log := r.Log.WithName("forceNNSRefresh").WithValues("node", name)
	log.Info("forcing NodeNetworkState refresh after NNCP applied")
	nns, err := r.readNNS(ctx, name)
	if err != nil {
		log.WithValues("error", err).
			Info("WARNING: failed retrieving NodeNetworkState to force refresh, it will be refreshed after regular period")
		return
	}
	if nns.Labels == nil {
		nns.Labels = map[string]string{}
	}
	nns.Labels[forceRefreshLabel] = fmt.Sprintf("%d", time.Now().UnixNano())

	err = r.Update(ctx, nns)
	if err != nil {
		log.WithValues("error", err).Info("WARNING: failed forcing NNS refresh, it will be refreshed after regular period")
	}
}

func (r *NodeNetworkConfigurationPolicyReconciler) readNNS(ctx context.Context, name string) (*nmstatev1beta1.NodeNetworkState, error) {
	nns := &nmstatev1beta1.NodeNetworkState{}
	err := r.Get(ctx, types.NamespacedName{Name: name}, nns)
	if err != nil {
		return nil, err
	}
	return nns, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) shouldAbortReconcile(
	ctx context.Context,
	instance *nmstatev1.NodeNetworkConfigurationPolicy,
) (bool, error) {
	logger := r.Log.WithName("shouldAbortReconcile")
	maxUnavailable, err := node.MaxUnavailableNodeCount(ctx, r.APIClient, instance)
	if err != nil {
		logger.Info("Error getting max unavailable count")
		return false, err
	}
	filter := enactmentconditions.LogicalConditionCountFilter{
		nmstateapi.NodeNetworkConfigurationEnactmentConditionFailing:     corev1.ConditionTrue,
		nmstateapi.NodeNetworkConfigurationEnactmentConditionProgressing: corev1.ConditionFalse,
	}

	failedConditionCount, err := enactmentconditions.CountConditionsLogicalAnd(ctx, r.APIClient, instance, filter)
	if err != nil {
		logger.Info("Error getting unavailable enactment count")
		return false, err
	}

	return failedConditionCount >= maxUnavailable, nil
}

func allPolicies(client client.Client, log logr.Logger) handler.TypedMapFunc[*corev1.Node, reconcile.Request] {
	return handler.TypedMapFunc[*corev1.Node, reconcile.Request](
		func(ctx context.Context, _ *corev1.Node) []reconcile.Request {
			logger := log.WithName("allPolicies")
			allPoliciesAsRequest := []reconcile.Request{}
			policyList := nmstatev1.NodeNetworkConfigurationPolicyList{}
			err := client.List(ctx, &policyList)
			if err != nil {
				logger.Error(err, "failed listing all NodeNetworkConfigurationPolicies to re-reconcile them after node created or updated")
				return []reconcile.Request{}
			}
			sort.Slice(policyList.Items, func(i, j int) bool {
				return policyList.Items[i].Name < policyList.Items[j].Name
			})
			for policyIndex := range policyList.Items {
				policy := policyList.Items[policyIndex]
				allPoliciesAsRequest = append(allPoliciesAsRequest, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: policy.Name,
					}})
			}
			return allPoliciesAsRequest
		})
}
