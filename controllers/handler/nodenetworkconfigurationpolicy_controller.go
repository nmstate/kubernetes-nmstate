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
	ReconcileFailed    = "ReconcileFailed"
	MaximumTimeBackoff = 30
	RetriesUntilFail   = 5
)

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
	nmstatectlShowFn = nmstatectl.Show
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
func (r *NodeNetworkConfigurationPolicyReconciler) Reconcile(_ context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("nodenetworkconfigurationpolicy", request.NamespacedName)

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1.NodeNetworkConfigurationPolicy{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Policy is not found, removing previous enactment if any")
			err = r.deleteEnactmentForPolicy(request.NamespacedName.Name)
			return ctrl.Result{}, err
		}
		log.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if !policyconditions.IsProgressing(&instance.Status.Conditions) {
		policyconditions.Reset(r.Client, request.NamespacedName)
	}

	// Policy conditions will be updated at the end so updating it
	// does not impact at applying state, it will increase just
	// reconcile time.
	defer policyconditions.Update(r.Client, r.APIClient, request.NamespacedName)

	policySelectors := selectors.NewFromPolicy(r.Client, instance)
	unmatchingNodeLabels, err := policySelectors.UnmatchedNodeLabels(nodeName)
	if err != nil {
		log.Error(err, "failed checking node selectors")
		return ctrl.Result{}, err
	}

	if len(unmatchingNodeLabels) > 0 {
		log.Info("Policy node selectors does not match node, removing previous enactment if any")
		err = r.deleteEnactmentForPolicy(request.NamespacedName.Name)
		return ctrl.Result{}, err
	}

	enactmentInstance, err := r.initializeEnactment(instance)
	if err != nil {
		log.Error(err, "Error initializing enactment")
		return ctrl.Result{}, err
	}
	previousConditions := &enactmentInstance.Status.Conditions
	enactmentConditions := enactmentconditions.New(r.APIClient, nmstateapi.EnactmentKey(nodeName, instance.Name))

	err = r.fillInEnactmentStatus(instance, enactmentInstance, enactmentConditions)
	if err != nil {
		log.Error(err, "failed filling in the NNCE status")
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	enactmentInstance, err = r.enactmentForPolicy(instance)
	if err != nil {
		log.Error(err, "error getting enactment for policy")
		return ctrl.Result{}, err
	}

	generationKey := strconv.FormatInt(enactmentInstance.Status.PolicyGeneration, 10)

	if r.shouldIncrementUnavailableNodeCount(previousConditions) {
		err = r.incrementUnavailableNodeCount(instance, generationKey)
		if err != nil {
			if apierrors.IsConflict(err) || errors.Is(err, node.MaxUnavailableLimitReachedError{}) {
				enactmentConditions.NotifyPending()
				log.Info(err.Error())
				shouldAbortEnactment, err := r.shouldAbortReconcile(instance)
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
					enactmentConditions.NotifyAborted(fmt.Errorf("reconciliation of enactment %q has aborted", enactmentInstance.Name))
					return ctrl.Result{}, nil
				}
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	enactmentConditions.NotifyProgressing()
	if policyconditions.IsUnknown(&instance.Status.Conditions) {
		policyconditions.Update(r.Client, r.APIClient, request.NamespacedName)
	}

	nmstateOutput, err := nmstate.ApplyDesiredState(r.APIClient, enactmentInstance.Status.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy on node %s at desired state apply: %q,\n %v",
			nodeName, nmstateOutput, err)
		log.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))
		err := r.incrementNNCERetryCount(instance, enactmentInstance, generationKey)
		if err != nil {
			log.Info("Error incrementing NNCERetry count")
			return ctrl.Result{}, err
		}

		if enactmentInstance.Status.RetryCount[generationKey] >= RetriesUntilFail {
			enactmentConditions.NotifyFailedToConfigure(errmsg)
			if r.Recorder != nil {
				r.Recorder.Event(instance,
					corev1.EventTypeWarning,
					ReconcileFailed,
					fmt.Errorf(
						"reconciliation of enactment %q has failed after %d retries",
						enactmentInstance.Name, RetriesUntilFail).Error())
			}
			return ctrl.Result{}, nil
		}
		enactmentConditions.NotifyRetrying(
			fmt.Errorf("failed to reconcile NodeNetworkConfigurationPolicy on node %s. Retrying %d/%d",
				nodeName,
				enactmentInstance.Status.RetryCount[generationKey]+1,
				RetriesUntilFail),
		)
		return ctrl.Result{Requeue: true}, nil
	}
	log.Info("nmstate", "output", nmstateOutput)

	enactmentConditions.NotifySuccess()
	if err := r.decrementUnavailableNodeCount(instance, generationKey); err != nil {
		r.Log.Info("Failed to update NNCP status, will retry", "error", err, "requeueAfter", "10s")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	r.forceNNSRefresh(nodeName)

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) incrementNNCERetryCount(
	instance *nmstatev1.NodeNetworkConfigurationPolicy,
	enactment *nmstatev1beta1.NodeNetworkConfigurationEnactment,
	generationKey string) error {
	if enactment.Status.RetryCount == nil {
		enactment.Status.RetryCount = map[string]int{}
	}
	count := enactment.Status.RetryCount[generationKey]

	enactment.Status.RetryCount[generationKey] = count + 1
	return enactmentstatus.Update(
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
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Second, time.Second*MaximumTimeBackoff),
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
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
) (*nmstatev1beta1.NodeNetworkConfigurationEnactment, error) {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	log := r.Log.WithName("initializeEnactment").WithValues("policy", policy.Name, "enactment", enactmentKey.Name)
	// Return if it's already initialize or we cannot retrieve it
	enactmentInstance := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.APIClient.Get(context.TODO(), enactmentKey, &enactmentInstance)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed getting enactment ")
	}
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("creating enactment")
		// Fetch the Node instance
		nodeInstance := &corev1.Node{}
		err = r.APIClient.Get(context.TODO(), types.NamespacedName{Name: nodeName}, nodeInstance)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting node")
		}
		enactmentInstance = nmstatev1beta1.NewEnactment(nodeInstance, policy)
		err = r.APIClient.Create(context.TODO(), &enactmentInstance)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating NodeNetworkConfigurationEnactment: %+v", enactmentInstance)
		}
		err = r.waitEnactmentCreated(enactmentKey)
		if err != nil {
			return nil, errors.Wrapf(err, "error waitting for NodeNetworkConfigurationEnactment: %+v", enactmentInstance)
		}
	} else {
		enactmentConditions := enactmentconditions.New(r.APIClient, enactmentKey)
		enactmentConditions.Reset()
	}

	return &enactmentInstance, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) fillInEnactmentStatus(
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
			r.APIClient,
			nmstateapi.EnactmentKey(nodeName, policy.Name),
			func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
				status.PolicyGeneration = policy.Generation
			},
		)
		if err2 != nil {
			return err2
		}
		enactmentConditions.NotifyGenerateFailure(err)
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
		r.APIClient,
		nmstateapi.EnactmentKey(nodeName, policy.Name),
		func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
			status.DesiredState = desiredStateWithDefaults
			status.CapturedStates = capturedStates
			status.PolicyGeneration = policy.Generation
			status.Features = features
		},
	)
}

func (r *NodeNetworkConfigurationPolicyReconciler) enactmentForPolicy(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
) (*nmstatev1beta1.NodeNetworkConfigurationEnactment, error) {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	instance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.APIClient.Get(context.TODO(), enactmentKey, instance)
	if err != nil {
		return nil, errors.Wrap(err, "getting enactment failed")
	}
	return instance, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) waitEnactmentCreated(enactmentKey types.NamespacedName) error {
	var enactmentInstance nmstatev1beta1.NodeNetworkConfigurationEnactment
	interval := time.Second
	timeout := 10 * time.Second
	pollErr := wait.PollUntilContextTimeout(context.TODO(), interval, timeout, true, /*immediate*/
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

func (r *NodeNetworkConfigurationPolicyReconciler) deleteEnactmentForPolicy(policyName string) error {
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
	err := r.APIClient.Delete(context.TODO(), &enactmentInstance)
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

func (r *NodeNetworkConfigurationPolicyReconciler) incrementUnavailableNodeCount(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	generationKey string) error {
	policyKey := types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}
	return retry.OnError(retry.DefaultRetry, func(error) bool { return true }, func() error {
		err := r.Client.Get(context.TODO(), policyKey, policy)
		if err != nil {
			return err
		}
		maxUnavailable, err := node.MaxUnavailableNodeCount(r.APIClient, policy)
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
		return r.Client.Status().Update(context.TODO(), policy)
	})
}

func (r *NodeNetworkConfigurationPolicyReconciler) decrementUnavailableNodeCount(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	generationKey string) error {
	policyKey := types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}
	err := tryDecrementingUnavailableNodeCount(r.Client, r.Client, policyKey, generationKey)
	if err != nil {
		r.Log.Error(err, "error decrementing unavailableNodeCount with cached client, trying again with non-cached client.")
		err = tryDecrementingUnavailableNodeCount(r.Client, r.APIClient, policyKey, generationKey)
		if err != nil {
			r.Log.Error(err, "error decrementing unavailableNodeCount with non-cached client")
			return err
		}
	}
	return nil
}

func tryDecrementingUnavailableNodeCount(
	statusWriterClient client.StatusClient,
	readerClient client.Reader,
	policyKey types.NamespacedName,
	generationKey string) error {
	instance := &nmstatev1.NodeNetworkConfigurationPolicy{}
	err := retry.OnError(retry.DefaultRetry, func(error) bool { return true }, func() error {
		err := readerClient.Get(context.TODO(), policyKey, instance)
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
		return statusWriterClient.Status().Update(context.TODO(), instance)
	})
	return err
}

func (r *NodeNetworkConfigurationPolicyReconciler) forceNNSRefresh(name string) {
	log := r.Log.WithName("forceNNSRefresh").WithValues("node", name)
	log.Info("forcing NodeNetworkState refresh after NNCP applied")
	nns, err := r.readNNS(name)
	if err != nil {
		log.WithValues("error", err).
			Info("WARNING: failed retrieving NodeNetworkState to force refresh, it will be refreshed after regular period")
		return
	}
	if nns.Labels == nil {
		nns.Labels = map[string]string{}
	}
	nns.Labels[forceRefreshLabel] = fmt.Sprintf("%d", time.Now().UnixNano())

	err = r.Client.Update(context.Background(), nns)
	if err != nil {
		log.WithValues("error", err).Info("WARNING: failed forcing NNS refresh, it will be refreshed after regular period")
	}
}

func (r *NodeNetworkConfigurationPolicyReconciler) readNNS(name string) (*nmstatev1beta1.NodeNetworkState, error) {
	nns := &nmstatev1beta1.NodeNetworkState{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name}, nns)
	if err != nil {
		return nil, err
	}
	return nns, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) shouldAbortReconcile(
	instance *nmstatev1.NodeNetworkConfigurationPolicy,
) (bool, error) {
	logger := r.Log.WithName("shouldAbortReconcile")
	maxUnavailable, err := node.MaxUnavailableNodeCount(r.APIClient, instance)
	if err != nil {
		logger.Info("Error getting max unavailable count")
		return false, err
	}
	filter := enactmentconditions.LogicalConditionCountFilter{
		nmstateapi.NodeNetworkConfigurationEnactmentConditionFailing:     corev1.ConditionTrue,
		nmstateapi.NodeNetworkConfigurationEnactmentConditionProgressing: corev1.ConditionFalse,
	}

	failedConditionCount, err := enactmentconditions.CountConditionsLogicalAnd(r.APIClient, instance, filter)
	if err != nil {
		logger.Info("Error getting unavailable enactment count")
		return false, err
	}

	return failedConditionCount >= maxUnavailable, nil
}

func allPolicies(client client.Client, log logr.Logger) handler.TypedMapFunc[*corev1.Node, reconcile.Request] {
	return handler.TypedMapFunc[*corev1.Node, reconcile.Request](
		func(context.Context, *corev1.Node) []reconcile.Request {
			logger := log.WithName("allPolicies")
			allPoliciesAsRequest := []reconcile.Request{}
			policyList := nmstatev1.NodeNetworkConfigurationPolicyList{}
			err := client.List(context.TODO(), &policyList)
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
