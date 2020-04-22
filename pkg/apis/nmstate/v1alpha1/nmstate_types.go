package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NMState is the Schema for the nmstates API
// +kubebuilder:resource:path=nmstates,scope=Cluster
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NMStateList contains a list of NMState
type NMStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NMState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NMState{}, &NMStateList{})
}
