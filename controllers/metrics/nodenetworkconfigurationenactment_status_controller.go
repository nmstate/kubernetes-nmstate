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
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
)

// NodeNetworkConfigurationEnactmentStatusReconciler reconciles NNCE objects for per-node status metrics
type NodeNetworkConfigurationEnactmentStatusReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	oldNodes map[string]struct{}
}

func (r *NodeNetworkConfigurationEnactmentStatusReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("metrics.nodenetworkconfigurationenactment_status", request.NamespacedName)
	log.Info("Reconcile")

	if err := r.reportStatistics(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed reporting NNCE status statistics: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationEnactmentStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.oldNodes = make(map[string]struct{})

	onConditionChange := conditionChangePredicate(func(obj client.Object) (shared.ConditionList, bool) {
		nnce, ok := obj.(*nmstatev1beta1.NodeNetworkConfigurationEnactment)
		if !ok {
			return nil, false
		}
		return nnce.Status.Conditions, true
	})

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkConfigurationEnactment{}).
		WithEventFilter(onConditionChange).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to add controller to NNCE status metrics Reconciler")
	}

	return nil
}

type enactmentStatusKey struct {
	node   string
	status string
}

func (r *NodeNetworkConfigurationEnactmentStatusReconciler) reportStatistics(ctx context.Context) error {
	nnceList := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	if err := r.List(ctx, &nnceList); err != nil {
		return err
	}

	counts := make(map[enactmentStatusKey]float64)
	newNodes := make(map[string]struct{})

	for i := range nnceList.Items {
		nodeName := nnceList.Items[i].Labels[shared.EnactmentNodeLabel]
		if nodeName == "" {
			continue
		}
		newNodes[nodeName] = struct{}{}

		status := activeConditionType(nnceList.Items[i].Status.Conditions)
		if status != "" {
			key := enactmentStatusKey{node: nodeName, status: string(status)}
			counts[key]++
		}
	}

	// Reset all known node+status combinations, then set current values
	for nodeName := range newNodes {
		for _, condType := range shared.NodeNetworkConfigurationEnactmentConditionTypes {
			key := enactmentStatusKey{node: nodeName, status: string(condType)}
			monitoring.EnactmentStatus.WithLabelValues(nodeName, string(condType)).Set(counts[key])
		}
	}

	// Delete metrics for nodes that no longer have any enactments
	for oldNode := range r.oldNodes {
		if _, exists := newNodes[oldNode]; !exists {
			for _, condType := range shared.NodeNetworkConfigurationEnactmentConditionTypes {
				monitoring.EnactmentStatus.Delete(prometheus.Labels{
					"node":   oldNode,
					"status": string(condType),
				})
			}
		}
	}

	r.oldNodes = newNodes

	return nil
}
