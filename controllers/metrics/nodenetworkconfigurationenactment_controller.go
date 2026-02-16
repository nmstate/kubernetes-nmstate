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
	"slices"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
)

// NodeNetworkConfigurationEnactment reconciles a NodeNetworkConfigurationEnactment object
type NodeNetworkConfigurationEnactmentReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	oldFeatures map[string]struct{}
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationEnactment object and calculate
// metrics with `nmstatectl stat`
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkConfigurationEnactmentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("metrics.nodenetworkconfigurationenactment", request.NamespacedName)
	log.Info("Reconcile")

	if err := r.reportStatistics(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed reporting statistics: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationEnactmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.oldFeatures = map[string]struct{}{}
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

			return !slices.Equal(oldNNCE.Status.Features, newNNCE.Status.Features)
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

	// Collect all unique feature names across all NNCEs
	newFeatures := make(map[string]struct{})
	for i := range nnceList.Items {
		for _, f := range nnceList.Items[i].Status.Features {
			newFeatures[f] = struct{}{}
		}
	}

	// Set gauge to 1 for every currently applied feature
	for f := range newFeatures {
		monitoring.AppliedFeatures.WithLabelValues(f).Set(1)
	}

	// Delete metrics for features that are no longer applied
	for f := range r.oldFeatures {
		if _, exists := newFeatures[f]; !exists {
			monitoring.AppliedFeatures.Delete(prometheus.Labels{"name": f})
		}
	}

	r.oldFeatures = newFeatures

	return nil
}
