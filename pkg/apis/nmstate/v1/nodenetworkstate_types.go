package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// State containes the namestatectl yaml [1] as string instead of golang struct
// so we don't need to be in sync with the schema.
//
// [1] https://github.com/nmstate/nmstate/blob/master/libnmstate/schemas/operational-state.yaml
// +k8s:openapi-gen=true
type State []byte

// NodeNetworkStateSpec defines the desired state of NodeNetworkState
// +k8s:openapi-gen=true
type NodeNetworkStateSpec struct {
	Managed bool `json:"managed"`
	// Name of the node reporting this state
	NodeName string `json:"nodeName"`
	// The desired configuration for the node
	DesiredState State `json:"desiredState,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkState is the Schema for the nodenetworkstates API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type NodeNetworkState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec         NodeNetworkStateSpec `json:"spec,omitempty"`
	CurrentState State                `json:"currentState,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkStateList contains a list of NodeNetworkState
type NodeNetworkStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeNetworkState{}, &NodeNetworkStateList{})
}
