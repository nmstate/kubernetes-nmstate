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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client"
	"github.com/nmstate/kubernetes-nmstate/pkg/nm"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	corev1 "k8s.io/api/core/v1"
)

// Added for test purposes
type NmstateUpdater func(
	ctx context.Context,
	client client.Client,
	node *corev1.Node,
	observedState shared.State,
	nns *nmstatev1beta1.NodeNetworkState,
	versions *nmstate.DependencyVersions,
) error
type NmstatectlShow func(ctx context.Context) (string, error)
type NmstatectlShowKernel func(ctx context.Context) error

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	lastState            shared.State
	nmstateUpdater       NmstateUpdater
	nmstatectlShow       NmstatectlShow
	nmstatectlShowKernel NmstatectlShowKernel
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	currentStateRaw, err := r.nmstatectlShow(ctx)
	if err != nil {
		return r.reportQueryFailure(ctx, request, err)
	}

	currentState, err := state.FilterOut(shared.NewState(currentStateRaw))
	if err != nil {
		return ctrl.Result{}, err
	}

	nnsInstance := &nmstatev1beta1.NodeNetworkState{}
	err = r.Get(ctx, request.NamespacedName, nnsInstance)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, errors.Wrap(err, "Failed to get nnstate")
		} else {
			nnsInstance = nil
		}
	}
	// Reduce apiserver hits by checking node's network state with last one
	if nnsInstance != nil && r.lastState.String() == currentState.String() {
		// The state did not change, but the query may have been failing
		// before, so make sure the conditions reflect the success.
		if err := r.reportQuerySuccess(ctx, nnsInstance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: node.NetworkStateRefreshWithJitter()}, nil
	} else {
		r.Log.Info("Creating/updating NodeNetworkState")
	}

	// Fetch the Node instance
	nodeInstance := &corev1.Node{}
	err = r.Get(ctx, request.NamespacedName, nodeInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	err = r.nmstateUpdater(ctx, r.Client, nodeInstance, currentState, nnsInstance, r.getDependencyVersions())
	if err != nil {
		err = errors.Wrap(err, "error at node reconcile creating NodeNetworkState")
		return ctrl.Result{}, err
	}

	// Cache currentState after successfully storing it at NodeNetworkState
	r.lastState = currentState

	// nmstateUpdater sets the conditions whenever it writes the state, but it
	// skips writing when the stored state is already up to date (e.g. after
	// a handler restart), so make sure the conditions reflect the success.
	if nnsInstance != nil {
		if err := r.reportQuerySuccess(ctx, nnsInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: node.NetworkStateRefreshWithJitter()}, nil
}

func (r *NodeReconciler) reportQuerySuccess(ctx context.Context, nns *nmstatev1beta1.NodeNetworkState) error {
	return nmstate.UpdateNodeNetworkStateQueryConditions(ctx, r.Client, nns,
		shared.NodeNetworkStateConditionQuerySucceeded, nmstate.QuerySucceededMessage)
}

// reportQueryFailure diagnoses a failed network state query and records it
// in the NodeNetworkState conditions, keeping the last successfully retrieved
// state. The query is retried at the normal refresh interval.
func (r *NodeReconciler) reportQueryFailure(ctx context.Context, request ctrl.Request, queryErr error) (ctrl.Result, error) {
	reason := shared.NodeNetworkStateConditionQueryFailed
	message := fmt.Sprintf("Failed to retrieve network state: %v", queryErr)
	if kernelErr := r.nmstatectlShowKernel(ctx); kernelErr == nil {
		reason = shared.NodeNetworkStateConditionNetworkManagerUnresponsive
		message = fmt.Sprintf("Failed to retrieve network state while kernel-only query works, "+
			"NetworkManager or D-Bus is not responding properly: %v", queryErr)
	} else {
		message = fmt.Sprintf("%s; kernel-only query also failed: %v", message, kernelErr)
	}
	r.Log.Error(queryErr, "Failed to retrieve network state", "reason", reason)

	nnsInstance := &nmstatev1beta1.NodeNetworkState{}
	err := r.Get(ctx, request.NamespacedName, nnsInstance)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, errors.Wrap(err, "Failed to get nnstate")
		}
		// Create an empty NNS so the failure is visible even if the state
		// could never be retrieved on this node.
		nodeInstance := &corev1.Node{}
		if err = r.Get(ctx, request.NamespacedName, nodeInstance); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		nnsInstance, err = nmstate.InitializeNodeNetworkState(ctx, r.Client, nodeInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := nmstate.UpdateNodeNetworkStateQueryConditions(ctx, r.Client, nnsInstance, reason, message); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: node.NetworkStateRefreshWithJitter()}, nil
}

func (r *NodeReconciler) getDependencyVersions() *nmstate.DependencyVersions {
	handlerNmstateVersion, err := nmstate.ExecuteCommand("nmstatectl", "--version")
	if err != nil {
		r.Log.Error(err, "failed retrieving handler nmstate version")
	}

	hostNetworkManagerVersion, err := nm.Version()
	if err != nil {
		r.Log.Error(err, "error retrieving host Networkmanager version")
	}

	return &nmstate.DependencyVersions{
		HandlerNmstateVersion:     handlerNmstateVersion,
		HostNetworkManagerVersion: hostNetworkManagerVersion,
	}
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.nmstateUpdater = nmstate.CreateOrUpdateNodeNetworkState
	r.nmstatectlShow = nmstatectl.ShowWithTimeout
	r.nmstatectlShowKernel = nmstatectl.ShowKernelLoopback

	// By default all this functors return true so controller watch all events,
	// but we only want to watch create/delete for current node.
	onCreationForThisNode := predicate.TypedFuncs[*corev1.Node]{
		CreateFunc: func(createEvent event.TypedCreateEvent[*corev1.Node]) bool {
			return node.EventIsForThisNode(createEvent.Object)
		},
		DeleteFunc: func(event.TypedDeleteEvent[*corev1.Node]) bool {
			return false
		},
		UpdateFunc: func(event.TypedUpdateEvent[*corev1.Node]) bool {
			return false
		},
		GenericFunc: func(event.TypedGenericEvent[*corev1.Node]) bool {
			return false
		},
	}

	// By default all this functors return true so controller watch all events,
	// but we only want to watch delete/update for current node.
	onDeleteOrForceUpdateForThisNode := predicate.TypedFuncs[*nmstatev1beta1.NodeNetworkState]{
		CreateFunc: func(createEvent event.TypedCreateEvent[*nmstatev1beta1.NodeNetworkState]) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.TypedDeleteEvent[*nmstatev1beta1.NodeNetworkState]) bool {
			return node.EventIsForThisNode(deleteEvent.Object)
		},
		UpdateFunc: func(updateEvent event.TypedUpdateEvent[*nmstatev1beta1.NodeNetworkState]) bool {
			return node.EventIsForThisNode(updateEvent.ObjectNew) &&
				shouldForceRefresh(updateEvent)
		},
		GenericFunc: func(event.TypedGenericEvent[*nmstatev1beta1.NodeNetworkState]) bool {
			return false
		},
	}

	c, err := controller.New("NodeNetworkState", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrap(err, "failed to create NodeNetworkState controller")
	}

	// Add watch for Node
	err = c.Watch(
		source.Kind(mgr.GetCache(), &corev1.Node{},
			&handler.TypedEnqueueRequestForObject[*corev1.Node]{},
			onCreationForThisNode,
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to add watch for Nodes")
	}

	// Add watch for NNS
	err = c.Watch(
		source.Kind(mgr.GetCache(),
			&nmstatev1beta1.NodeNetworkState{},
			handler.TypedEnqueueRequestForOwner[*nmstatev1beta1.NodeNetworkState](mgr.GetScheme(), mgr.GetRESTMapper(), &corev1.Node{}),
			onDeleteOrForceUpdateForThisNode,
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to add watch for NNSes")
	}

	return nil
}

func shouldForceRefresh(updateEvent event.TypedUpdateEvent[*nmstatev1beta1.NodeNetworkState]) bool {
	newForceRefresh, hasForceRefreshNow := updateEvent.ObjectNew.GetLabels()[forceRefreshLabel]
	if !hasForceRefreshNow {
		return false
	}
	oldForceRefresh, hasForceRefreshLabelPreviously := updateEvent.ObjectOld.GetLabels()[forceRefreshLabel]
	if !hasForceRefreshLabelPreviously {
		return true
	}
	return oldForceRefresh != newForceRefresh
}
