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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nmstates,scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type",description="Status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].reason",description="Reason"
// +kubebuilder:deprecatedversion:warning="nmstate/v1beta1 deprecated, use nmstate/v1 instead"

// NMState is the Schema for the nmstates API
type NMState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// We are adding a default value for the Spec field because we want it to get automatically
	// populated even if user does not specify it at all. By default, the k8s apiserver populates
	// defaults only if you define "spec: {}" in your CR, otherwise it ignores the spec tree.
	// Ref.: https://ahmet.im/blog/crd-generation-pitfalls/#defaulting-on-nested-structs

	// +kubebuilder:default:={}
	Spec   shared.NMStateSpec   `json:"spec,omitempty"`
	Status shared.NMStateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NMStateList contains a list of NMState
type NMStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NMState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NMState{}, &NMStateList{})
}
