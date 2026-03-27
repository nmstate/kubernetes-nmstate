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

package tls

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SecurityProfileWatcher watches the APIServer object for TLS profile changes
// and triggers a callback when the profile changes.
type SecurityProfileWatcher struct {
	client.Client

	// InitialTLSProfileSpec is the TLS profile spec that was configured when the operator started.
	InitialTLSProfileSpec TLSProfileSpec

	// OnProfileChange is called when the TLS profile changes.
	OnProfileChange func(ctx context.Context, oldTLSProfileSpec, newTLSProfileSpec TLSProfileSpec)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityProfileWatcher) SetupWithManager(mgr ctrl.Manager) error {
	apiServerObj := &unstructured.Unstructured{}
	apiServerObj.SetGroupVersionKind(apiServerGVK)

	if err := ctrl.NewControllerManagedBy(mgr).
		Named("tlssecurityprofilewatcher").
		For(apiServerObj, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return e.Object.GetName() == apiServerName
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return e.ObjectNew.GetName() == apiServerName
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return e.Object.GetName() == apiServerName
				},
				GenericFunc: func(e event.GenericEvent) bool {
					return e.Object.GetName() == apiServerName
				},
			},
		)).
		WithLogConstructor(func(_ *reconcile.Request) logr.Logger {
			return mgr.GetLogger().WithValues(
				"controller", "tlssecurityprofilewatcher",
			)
		}).
		Complete(r); err != nil {
		return fmt.Errorf("could not set up controller for TLS security profile watcher: %w", err)
	}

	return nil
}

// Reconcile watches for changes to the APIServer TLS profile.
func (r *SecurityProfileWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "name", req.Name)

	logger.V(1).Info("Reconciling APIServer TLS profile")
	defer logger.V(1).Info("Finished reconciling APIServer TLS profile")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(apiServerGVK)

	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get APIServer %s: %w", req.String(), err)
	}

	profile, err := parseTLSSecurityProfile(obj.Object)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse TLS profile from APIServer %s: %w", req.String(), err)
	}

	currentTLSProfileSpec, err := GetTLSProfileSpec(profile)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get TLS profile from APIServer %s: %w", req.String(), err)
	}

	if !reflect.DeepEqual(r.InitialTLSProfileSpec, currentTLSProfileSpec) {
		if r.OnProfileChange != nil {
			r.OnProfileChange(ctx, r.InitialTLSProfileSpec, currentTLSProfileSpec)
		}
		r.InitialTLSProfileSpec = currentTLSProfileSpec
	}

	return ctrl.Result{}, nil
}
