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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
	"github.com/nmstate/kubernetes-nmstate/pkg/policyconditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/selectors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	nodeName                   string
	nodeConfigurationTimeout   = 5 * time.Minute
	nodeRunningUpdateRetryTime = 5 * time.Second
	watchPredicate             = predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			// [1] https://blog.openshift.com/kubernetes-operators-best-practices/
			generationIsDifferent := updateEvent.MetaNew.GetGeneration() != updateEvent.MetaOld.GetGeneration()
			return generationIsDifferent
		},
	}
)

func init() {
	if !environment.IsHandler() {
		return
	}

	nodeName = environment.NodeName()
	if len(nodeName) == 0 {
		panic("NODE_NAME is mandatory")
	}
}

// NodeNetworkConfigurationPolicyReconciler reconciles a NodeNetworkConfigurationPolicy object
type NodeNetworkConfigurationPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NodeNetworkConfigurationPolicyReconciler) waitEnactmentCreated(enactmentKey types.NamespacedName) error {
	var enactment nmstatev1beta1.NodeNetworkConfigurationEnactment
	pollErr := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		err := r.Client.Get(context.TODO(), enactmentKey, &enactment)
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

func (r *NodeNetworkConfigurationPolicyReconciler) initializeEnactment(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) error {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	log := r.Log.WithName("initializeEnactment").WithValues("policy", policy.Name, "enactment", enactmentKey.Name)
	// Return if it's already initialize or we cannot retrieve it
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.Client.Get(context.TODO(), enactmentKey, &enactment)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed getting enactment ")
	}
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("creating enactment")
		enactment = nmstatev1beta1.NewEnactment(nodeName, policy)
		err = r.Client.Create(context.TODO(), &enactment)
		if err != nil {
			return errors.Wrapf(err, "error creating NodeNetworkConfigurationEnactment: %+v", enactment)
		}
		err = r.waitEnactmentCreated(enactmentKey)
		if err != nil {
			return errors.Wrapf(err, "error waitting for NodeNetworkConfigurationEnactment: %+v", enactment)
		}
	} else {
		enactmentConditions := enactmentconditions.New(r.Client, enactmentKey)
		enactmentConditions.Reset()
	}

	return enactmentstatus.Update(r.Client, enactmentKey, func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
		status.DesiredState = policy.Spec.DesiredState
		status.PolicyGeneration = policy.Generation
	})
}

func (r *NodeNetworkConfigurationPolicyReconciler) getEnactmentCount(policy *nmstatev1beta1.NodeNetworkConfigurationPolicy) (enactmentconditions.ConditionCount, error) {
	enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	policyLabelFilter := client.MatchingLabels{nmstateapi.EnactmentPolicyLabel: policy.GetName()}
	err := r.Client.List(context.TODO(), &enactments, policyLabelFilter)
	if err != nil {
		return nil, errors.Wrap(err, "getting enactment list failed")
	}
	enactmentCount := enactmentconditions.Count(enactments, policy.Generation)
	return enactmentCount, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) setNodeRunningUpdate(policyKey types.NamespacedName) error {
	policy := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	err := r.Client.Get(context.TODO(), policyKey, policy)
	if err != nil {
		return err
	}
	if policy.Status.NodeRunningUpdate != "" {
		return fmt.Errorf("another node is working on configuration")
	}
	policy.Status.NodeRunningUpdate = nodeName
	policy.Status.NodeUpdateStart = &metav1.Time{Time: time.Now()}
	err = r.Client.Status().Update(context.TODO(), policy)
	if err != nil {
		return err
	}
	return nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) claimNodeRunningUpdate(
	policy *nmstatev1beta1.NodeNetworkConfigurationPolicy,
	ec *enactmentconditions.EnactmentConditions,
) (ctrl.Result, error) {
	if policy.Status.NodeRunningUpdate != "" && metav1.Now().Sub(policy.Status.NodeUpdateStart.Time) > nodeConfigurationTimeout {
		// a node has been running the update for too long
		errmsg := fmt.Errorf("A node has been configuring for too long, aborting")
		ec.NotifyFailedToConfigure(errmsg)
		return reconcile.Result{}, errmsg
	}
	err := r.setNodeRunningUpdate(types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()})
	if err != nil {
		return ctrl.Result{RequeueAfter: nodeRunningUpdateRetryTime}, err
	}
	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) releaseNodeRunningUpdate(policyKey types.NamespacedName) {
	instance := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), policyKey, instance)
		if err != nil {
			r.Log.Info("Failed to get policy, retrying")
			return err
		}
		// release only if we are the owner
		if instance.Status.NodeRunningUpdate != nodeName {
			return nil
		}

		instance.Status.NodeRunningUpdate = ""
		instance.Status.NodeUpdateStart = nil

		err = r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			if apierrors.IsConflict(err) {
				r.Log.Info("Failed to update policy status, retrying")
			} else {
				r.Log.Error(err, "Failed to release NodeRunningUpdate")
			}
			return err
		}
		return nil
	})
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationPolicy object and makes changes based on the state read
// and what is in the NodeNetworkConfigurationPolicy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkConfigurationPolicyReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("nodenetworkconfigurationpolicy", request.NamespacedName)

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	policyconditions.Reset(r.Client, request.NamespacedName)

	err = r.initializeEnactment(*instance)
	if err != nil {
		log.Error(err, "Error initializing enactment")
	}

	enactmentConditions := enactmentconditions.New(r.Client, nmstateapi.EnactmentKey(nodeName, instance.Name))

	if !instance.Spec.Parallel {
		defer r.releaseNodeRunningUpdate(request.NamespacedName)
	}
	// Policy conditions will be updated at the end so updating it
	// does not impact at applying state, it will increase just
	// reconcile time.
	defer policyconditions.Update(r.Client, request.NamespacedName)

	policySelectors := selectors.NewFromPolicy(r.Client, *instance)
	unmatchingNodeLabels, err := policySelectors.UnmatchedNodeLabels(nodeName)
	if err != nil {
		log.Error(err, "failed checking node selectors")
		enactmentConditions.NotifyNodeSelectorFailure(err)
	}
	if len(unmatchingNodeLabels) > 0 {
		log.Info("Policy node selectors does not match node")
		enactmentConditions.NotifyNodeSelectorNotMatching(unmatchingNodeLabels)
		return ctrl.Result{}, nil
	}

	enactmentConditions.NotifyMatching()

	if !instance.Spec.Parallel {
		res, err := r.claimNodeRunningUpdate(instance, &enactmentConditions)
		if err != nil {
			return res, err
		}
	}

	enactmentConditions.NotifyProgressing()
	nmstateOutput, err := nmstate.ApplyDesiredState(r.Client, instance.Spec.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy at desired state apply: %s, %v", nmstateOutput, err)

		enactmentConditions.NotifyFailedToConfigure(errmsg)
		log.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))

		return ctrl.Result{}, nil
	}
	log.Info("nmstate", "output", nmstateOutput)

	enactmentConditions.NotifySuccess()

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkConfigurationPolicy{}).
		WithEventFilter(watchPredicate).
		Complete(r)
}

func desiredState(object runtime.Object) (nmstateapi.State, error) {
	var state nmstateapi.State
	switch v := object.(type) {
	default:
		return nmstateapi.State{}, fmt.Errorf("unexpected type %T", v)
	case *nmstatev1beta1.NodeNetworkConfigurationPolicy:
		state = v.Spec.DesiredState
	}
	return state, nil
}
