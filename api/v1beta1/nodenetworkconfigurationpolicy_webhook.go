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

package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
)

// log is for logging in this package.
var nodenetworkconfigurationpolicylog = logf.Log.WithName("nodenetworkconfigurationpolicy-resource")

func (r *NodeNetworkConfigurationPolicy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nmstate-nmstate-io-v1beta1-nodenetworkconfigurationpolicy,mutating=true,failurePolicy=fail,groups=nmstate.nmstate.io,resources=nodenetworkconfigurationpolicies,verbs=create;update,versions=v1beta1,v1beta1,name=mnodenetworkconfigurationpolicy.kb.io

var _ webhook.Defaulter = &NodeNetworkConfigurationPolicy{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NodeNetworkConfigurationPolicy) Default() {
	v1alpha1Policy := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
	r.ConvertTo(v1alpha1Policy)
	v1alpha1Policy.Default()
	r.ConvertFrom(v1alpha1Policy)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-nmstate-nmstate-io-v1beta1-nodenetworkconfigurationpolicy,mutating=false,failurePolicy=fail,groups=nmstate.nmstate.io,resources=nodenetworkconfigurationpolicies,versions=v1beta1,v1beta1,name=vnodenetworkconfigurationpolicy.kb.io

var _ webhook.Validator = &NodeNetworkConfigurationPolicy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeNetworkConfigurationPolicy) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeNetworkConfigurationPolicy) ValidateUpdate(old runtime.Object) error {
	oldPolicy, ok := old.(*NodeNetworkConfigurationPolicy)
	if !ok {
		return fmt.Errorf("cannot convert to NodeNetworkConfigurationPolicy")
	}

	v1alpha1Policy := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
	v1alpha1OldPolicy := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
	r.ConvertTo(v1alpha1Policy)
	oldPolicy.ConvertTo(v1alpha1OldPolicy)
	return v1alpha1Policy.ValidateUpdate(v1alpha1OldPolicy)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeNetworkConfigurationPolicy) ValidateDelete() error {
	return nil
}
