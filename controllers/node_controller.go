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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	corev1 "k8s.io/api/core/v1"
)

// Added for test purposes
type NmstateUpdater func(client client.Client, node *corev1.Node, namespace client.ObjectKey, observedStateRaw string) error
type NmstatectlShow func() (string, error)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	lastState      string
	nmstateUpdater NmstateUpdater
	nmstatectlShow NmstatectlShow
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	currentState, err := r.nmstatectlShow()
	if err != nil {
		// We cannot call nmstatectl show let's reconcile again
		return ctrl.Result{}, err
	}

	// Reduce apiserver hits by checking node's network state with last one
	if r.lastState != "" && !r.networkStateChanged(currentState) {
		return ctrl.Result{RequeueAfter: node.NetworkStateRefresh}, err
	} else {
		r.Log.Info("Network configuration changed, updating NodeNetworkState")
	}

	// Fetch the Node instance
	instance := &corev1.Node{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, instance)
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
	err = r.nmstateUpdater(r.Client, instance, request.NamespacedName, currentState)
	if err != nil {
		err = errors.Wrap(err, "error at node reconcile creating NodeNetworkState")
		return ctrl.Result{}, err
	}

	// Cache currentState after successfully storing it at NodeNetworkState
	r.lastState = currentState

	return ctrl.Result{RequeueAfter: node.NetworkStateRefresh}, nil
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.nmstateUpdater = nmstate.CreateOrUpdateNodeNetworkState
	r.nmstatectlShow = nmstatectl.Show

	// By default all this functors return true so controller watch all events,
	// but we only want to watch create/delete for current node.
	onCreationForThisNode := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return nmstate.EventIsForThisNode(createEvent.Meta)
		},
		DeleteFunc: func(event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(onCreationForThisNode).
		Complete(r)
}

func (r *NodeReconciler) networkStateChanged(currentState string) bool {
	return state.RemoveDynamicAttributes(r.lastState) != state.RemoveDynamicAttributes(currentState)
}
