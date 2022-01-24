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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactment"
)

// NodeReconciler reconciles a Node object
type NodeNetworkConfigurationEnactmentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationEnactment object and makes cleanup
// if needed.
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkConfigurationEnactmentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nodenetworkconfigurationenactment", request.NamespacedName)

	// Fetch the NodeNetworkConfigurationEnactment instance
	enactmentInstance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, enactmentInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error retrieving enactment")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	policyName := enactmentInstance.Labels[shared.EnactmentPolicyLabel]
	policyInstance := &nmstatev1.NodeNetworkConfigurationPolicy{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: policyName}, policyInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Policy is not found, removing the enactment")
			err = r.Client.Delete(context.TODO(), enactmentInstance)
			return ctrl.Result{}, err
		}
		log.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: enactment.RefreshWithJitter()}, nil
}

func (r *NodeNetworkConfigurationEnactmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// By default all this functors return true so controller watch all events,
	// but we only want to watch create for current node.
	onCreationForThisEnactment := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
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

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkConfigurationEnactment{}).
		WithEventFilter(onCreationForThisEnactment).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to add controller to NNCE Reconciler")
	}

	return nil
}
