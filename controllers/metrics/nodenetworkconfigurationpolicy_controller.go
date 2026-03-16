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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
)

// NodeNetworkConfigurationPolicyReconciler reconciles NNCP objects for status metrics
type NodeNetworkConfigurationPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NodeNetworkConfigurationPolicyReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("metrics.nodenetworkconfigurationpolicy", request.NamespacedName)
	log.Info("Reconcile")

	if err := r.reportStatistics(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed reporting NNCP status statistics: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	onConditionChange := predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNNCP, ok := e.ObjectOld.(*nmstatev1.NodeNetworkConfigurationPolicy)
			if !ok {
				return true
			}
			newNNCP, ok := e.ObjectNew.(*nmstatev1.NodeNetworkConfigurationPolicy)
			if !ok {
				return true
			}
			return activeConditionType(oldNNCP.Status.Conditions) != activeConditionType(newNNCP.Status.Conditions)
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1.NodeNetworkConfigurationPolicy{}).
		WithEventFilter(onConditionChange).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to add controller to NNCP status metrics Reconciler")
	}

	return nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) reportStatistics(ctx context.Context) error {
	nncpList := nmstatev1.NodeNetworkConfigurationPolicyList{}
	if err := r.List(ctx, &nncpList); err != nil {
		return err
	}

	counts := make(map[shared.ConditionType]float64)
	for _, condType := range shared.NodeNetworkConfigurationPolicyConditionTypes {
		counts[condType] = 0
	}

	for i := range nncpList.Items {
		status := activeConditionType(nncpList.Items[i].Status.Conditions)
		if status != "" {
			counts[status]++
		}
	}

	for condType, count := range counts {
		monitoring.PolicyStatus.WithLabelValues(string(condType)).Set(count)
	}

	return nil
}
