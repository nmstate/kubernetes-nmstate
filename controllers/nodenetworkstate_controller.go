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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	corev1 "k8s.io/api/core/v1"
)

// NodeNetworkStateReconciler reconciles a NodeNetworkState object
type NodeNetworkStateReconciler struct {
	client.Client
	Config          *rest.Config
	Log             logr.Logger
	Scheme          *runtime.Scheme
	lastStateByNode map[string]shared.State
}

type CurrentStateReconciler struct {
	client.Client
	Config          *rest.Config
	Log             logr.Logger
	Scheme          *runtime.Scheme
	lastStateByNode map[string]shared.State
}

// Reconcile reads that state of the cluster for a NodeNetworkState object and makes changes based on the state read
// and what is in the NodeNetworkState.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *CurrentStateReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {

	//currentStateRaw, err := r.nmstatectlShow()
	currentStateRaw, err := nmstatectl.ShowAtNode(r.Config, request.Name)
	if err != nil {
		// We cannot call nmstatectl show let's reconcile again
		return ctrl.Result{}, err
	}

	currentState, err := state.FilterOut(shared.NewState(currentStateRaw))
	if err != nil {
		return ctrl.Result{}, err
	}

	lastState, hasLastState := r.lastStateByNode[request.Name]

	// Reduce apiserver hits by checking node's network state with last one
	if hasLastState && lastState.String() == currentState.String() {
		return ctrl.Result{RequeueAfter: nmstatenode.NetworkStateRefreshWithJitter()}, err
	} else {
		r.Log.Info("Network configuration changed, updating NodeNetworkState")
	}

	// Fetch the Node instance
	node := &corev1.Node{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Node is gone the NodeNetworkState delete event has being
			// triggered by k8s garbage collector we don't need to
			// re-create the NodeNetworkState
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	nmstate.CreateOrUpdateNodeNetworkState(r.Client, node, request.NamespacedName, currentState)
	if err != nil {
		err = errors.Wrap(err, "error at node reconcile creating NodeNetworkStateNetworkState")
		return ctrl.Result{}, err
	}

	// Cache currentState after successfully storing it at NodeNetworkState
	r.lastStateByNode[request.Name] = currentState

	return ctrl.Result{RequeueAfter: nmstatenode.NetworkStateRefreshWithJitter()}, nil
}

// Reconcile reads that state of the cluster for a NodeNetworkState object and makes changes based on the state read
// and what is in the NodeNetworkState.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkStateReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {

	node := &corev1.Node{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Node is gone the NodeNetworkState delete event has being
			// triggered by k8s garbage collector we don't need to
			// re-create the NodeNetworkState
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	//currentStateRaw, err := r.nmstatectlShow()
	currentStateRaw, err := nmstatectl.ShowAtNode(r.Config, request.Name)
	if err != nil {
		// We cannot call nmstatectl show let's reconcile again
		return ctrl.Result{}, err
	}

	currentState, err := state.FilterOut(shared.NewState(currentStateRaw))
	if err != nil {
		return ctrl.Result{}, err
	}

	nmstate.CreateOrUpdateNodeNetworkState(r.Client, node, request.NamespacedName, currentState)
	if err != nil {
		err = errors.Wrap(err, "error at node reconcile creating NodeNetworkStateNetworkState")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NodeNetworkStateReconciler) SetupWithManager(mgr ctrl.Manager) error {

	csReconciler := &CurrentStateReconciler{
		Client:          r.Client,
		Config:          r.Config,
		Log:             r.Log,
		Scheme:          r.Scheme,
		lastStateByNode: map[string]shared.State{},
	}

	onDelete := predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	onCreateOrForceRefresh := predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return shouldForceRefresh(updateEvent)
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkState{}).
		WithEventFilter(onDelete).
		Complete(r)
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkState{}).
		WithEventFilter(onCreateOrForceRefresh).
		Complete(csReconciler)
	if err != nil {
		return err
	}

	return nil
}

func shouldForceRefresh(updateEvent event.UpdateEvent) bool {
	newForceRefresh, hasForceRefreshNow := updateEvent.MetaNew.GetLabels()[forceNNSRefreshLabel]
	if !hasForceRefreshNow {
		return false
	}
	oldForceRefresh, hasForceRefreshLabelPreviously := updateEvent.MetaOld.GetLabels()[forceNNSRefreshLabel]
	if !hasForceRefreshLabelPreviously {
		return true
	}
	return oldForceRefresh != newForceRefresh
}
