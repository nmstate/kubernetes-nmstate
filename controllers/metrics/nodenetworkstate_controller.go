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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
)

// NodeNetworkStateReconciler reconciles a NodeNetworkState object for metrics
type NodeNetworkStateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Track interface types per node to clean up stale metrics
	oldInterfaceTypes map[string]map[string]struct{} // node name -> set of interface types
	// Track route keys per node to clean up stale metrics
	oldRouteKeys map[string]map[state.RouteKey]struct{} // node name -> set of route keys
}

// Reconcile reads the state of the cluster for a NodeNetworkState object and calculates
// metrics for network interface counts by type and node.
func (r *NodeNetworkStateReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("metrics.nodenetworkstate", request.NamespacedName)
	log.Info("Reconcile")

	nodeName := request.Name

	nnsInstance := &nmstatev1beta1.NodeNetworkState{}
	err := r.Client.Get(ctx, request.NamespacedName, nnsInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// NNS has been deleted, clean up metrics for this node
			r.deleteNodeMetrics(nodeName)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error retrieving NodeNetworkState")
		return ctrl.Result{}, err
	}

	// Count interfaces by type for this node
	counts, err := state.CountInterfacesByType(nnsInstance.Status.CurrentState)
	if err != nil {
		log.Error(err, "Failed to count interfaces by type")
		return ctrl.Result{}, err
	}

	// Update interface metrics for this node
	r.updateNodeInterfaceMetrics(nodeName, counts)

	// Count routes by IP stack and type for this node
	routeCounts, err := state.CountRoutes(nnsInstance.Status.CurrentState)
	if err != nil {
		log.Error(err, "Failed to count routes")
		return ctrl.Result{}, err
	}

	// Update route metrics for this node
	r.updateNodeRouteMetrics(nodeName, routeCounts)

	return ctrl.Result{}, nil
}

func (r *NodeNetworkStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.oldInterfaceTypes = make(map[string]map[string]struct{})
	r.oldRouteKeys = make(map[string]map[state.RouteKey]struct{})

	onCreationOrUpdateForThisNNS := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNNS, ok := e.ObjectOld.(*nmstatev1beta1.NodeNetworkState)
			if !ok {
				return false
			}
			newNNS, ok := e.ObjectNew.(*nmstatev1beta1.NodeNetworkState)
			if !ok {
				return false
			}

			// Reconcile if the current state has changed
			return oldNNS.Status.CurrentState.String() != newNNS.Status.CurrentState.String()
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkState{}).
		WithEventFilter(onCreationOrUpdateForThisNNS).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "failed to add controller to NNS metrics Reconciler")
	}

	return nil
}

// updateNodeInterfaceMetrics sets the interface count metrics for a specific node
func (r *NodeNetworkStateReconciler) updateNodeInterfaceMetrics(nodeName string, counts map[string]int) {
	// Get the old interface types for this node to detect removed types
	oldTypes := r.oldInterfaceTypes[nodeName]
	newTypes := make(map[string]struct{})

	// Set metrics for current interface types
	for ifaceType, count := range counts {
		monitoring.NetworkInterfaces.With(prometheus.Labels{
			"type": ifaceType,
			"node": nodeName,
		}).Set(float64(count))
		newTypes[ifaceType] = struct{}{}
	}

	// Delete metrics for interface types that no longer exist on this node
	for oldType := range oldTypes {
		if _, exists := newTypes[oldType]; !exists {
			monitoring.NetworkInterfaces.Delete(prometheus.Labels{
				"type": oldType,
				"node": nodeName,
			})
		}
	}

	// Store current types for next reconcile
	r.oldInterfaceTypes[nodeName] = newTypes
}

// updateNodeRouteMetrics sets the route count metrics for a specific node
func (r *NodeNetworkStateReconciler) updateNodeRouteMetrics(nodeName string, counts map[state.RouteKey]int) {
	// Get the old route keys for this node to detect removed keys
	oldKeys := r.oldRouteKeys[nodeName]
	newKeys := make(map[state.RouteKey]struct{})

	// Set metrics for current route keys
	for key, count := range counts {
		monitoring.NetworkRoutes.With(prometheus.Labels{
			"node":     nodeName,
			"ip_stack": key.IPStack,
			"type":     key.Type,
		}).Set(float64(count))
		newKeys[key] = struct{}{}
	}

	// Delete metrics for route keys that no longer exist on this node
	for oldKey := range oldKeys {
		if _, exists := newKeys[oldKey]; !exists {
			monitoring.NetworkRoutes.Delete(prometheus.Labels{
				"node":     nodeName,
				"ip_stack": oldKey.IPStack,
				"type":     oldKey.Type,
			})
		}
	}

	// Store current keys for next reconcile
	r.oldRouteKeys[nodeName] = newKeys
}

// deleteNodeMetrics removes all interface and route count metrics for a specific node
func (r *NodeNetworkStateReconciler) deleteNodeMetrics(nodeName string) {
	// Delete interface metrics
	if oldTypes, ok := r.oldInterfaceTypes[nodeName]; ok {
		for ifaceType := range oldTypes {
			monitoring.NetworkInterfaces.Delete(prometheus.Labels{
				"type": ifaceType,
				"node": nodeName,
			})
		}
		delete(r.oldInterfaceTypes, nodeName)
	}

	// Delete route metrics
	if oldKeys, ok := r.oldRouteKeys[nodeName]; ok {
		for key := range oldKeys {
			monitoring.NetworkRoutes.Delete(prometheus.Labels{
				"node":     nodeName,
				"ip_stack": key.IPStack,
				"type":     key.Type,
			})
		}
		delete(r.oldRouteKeys, nodeName)
	}
}
