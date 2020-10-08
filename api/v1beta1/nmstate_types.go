/*
Copyright The Kubernetes Authors.

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
)

// +kubebuilder:object:root=true

// NMState is the Schema for the nmstates API
// +kubebuilder:resource:path=nmstates,scope=Cluster
// +kubebuilder:storageversion
type NMState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NMStateSpec `json:"spec,omitempty"`
}

// NMStateSpec defines the desired state of NMstate
type NMStateSpec struct {
	// NodeSelector is an optional selector that will be added to handler DaemonSet manifest
	// for both workers and masters (https://github.com/nmstate/kubernetes-nmstate/blob/master/deploy/handler/operator.yaml).
	// If NodeSelector is specified, the handler will run only on nodes that have each of the indicated key-value pairs
	// as labels applied to the node.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
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
