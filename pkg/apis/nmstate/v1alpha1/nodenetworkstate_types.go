package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// State containes the namestatectl yaml [1] as string instead of golang struct
// so we don't need to be in sync with the schema.
//
// [1] https://github.com/nmstate/nmstate/blob/master/libnmstate/schemas/operational-state.yaml
// +k8s:openapi-gen=true
type State []byte

// NodeNetworkStateStatus is the status of the NodeNetworkState of a specific node
// +k8s:openapi-gen=true
type NodeNetworkStateStatus struct {
	CurrentState State `json:"currentState,omitempty"`

	Conditions ConditionList `json:"conditions,omitempty" optional:"true"`
}

const (
	NodeNetworkStateConditionAvailable ConditionType = "Available"
	NodeNetworkStateConditionFailing   ConditionType = "Failing"
)

const (
	NodeNetworkStateConditionFailedToConfigure      ConditionReason = "FailedToConfigure"
	NodeNetworkStateConditionSuccessfullyConfigured ConditionReason = "SuccessfullyConfigured"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkState is the Schema for the nodenetworkstates API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkstates,shortName=nns
type NodeNetworkState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status NodeNetworkStateStatus `json:"status,omitempty"`
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
