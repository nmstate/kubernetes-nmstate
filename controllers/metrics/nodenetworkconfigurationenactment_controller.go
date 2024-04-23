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

package metrics

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

// NodeNetworkConfigurationEnactment reconciles a NodeNetworkConfigurationEnactment object
type NodeNetworkConfigurationEnactmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	oldNNCEs map[string]*nmstatev1beta1.NodeNetworkConfigurationEnactment
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationEnactment object and calculate
// metrics with `nmstatectl stat`
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkConfigurationEnactmentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("metrics.nodenetworkconfigurationenactment", request.NamespacedName)
	log.Info("Reconcile")

	enactmentInstance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, enactmentInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// NNCE has being delete let's clean the old NNCEs map
			delete(r.oldNNCEs, request.Name)

			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error retrieving enactment")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if err := r.reportStatistics(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed reporting statistics: %w", err)
	}

	// After reporting metrics store this NNCE as old to calculate gaugue
	r.oldNNCEs[enactmentInstance.Name] = enactmentInstance

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationEnactmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.oldNNCEs = map[string]*nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	// By default all this functors return true so controller watch all events,
	// but we only want to watch create for current node.
	onCreationOrUpdateForThisEnactment := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNNCE, ok := e.ObjectOld.(*nmstatev1beta1.NodeNetworkConfigurationEnactment)
			if !ok {
				return false
			}
			newNNCE, ok := e.ObjectNew.(*nmstatev1beta1.NodeNetworkConfigurationEnactment)
			if !ok {
				return false
			}

			return !reflect.DeepEqual(oldNNCE.Status.Features, newNNCE.Status.Features)
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkConfigurationEnactment{}).
		WithEventFilter(onCreationOrUpdateForThisEnactment).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to add controller to NNCE metrics Reconciler")
	}

	return nil
}

func (r *NodeNetworkConfigurationEnactmentReconciler) reportStatistics(ctx context.Context) error {
	nnceList := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	if err := r.List(ctx, &nnceList); err != nil {
		return err
	}

	// Calculate old and new cluster wide features
	oldFeatures := []string{}
	newFeatures := []string{}
	for i := range nnceList.Items {
		newFeatures = append(newFeatures, nnceList.Items[i].Status.Features...)
		oldNNCE, ok := r.oldNNCEs[nnceList.Items[i].Name]
		if ok {
			oldFeatures = append(oldFeatures, oldNNCE.Status.Features...)
		}
	}

	oldStats := nmstatectl.NewStats(oldFeatures)
	newStats := nmstatectl.NewStats(newFeatures)

	statsToInc := newStats.Subtract(oldStats)
	for f := range statsToInc.Features {
		monitoring.AppliedFeatures.WithLabelValues(f).Inc()
	}

	statsToDel := oldStats.Subtract(newStats)
	for f := range statsToDel.Features {
		monitoring.AppliedFeatures.WithLabelValues(f).Dec()
	}
	return nil
}
