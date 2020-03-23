package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NMstate is the Schema for the nmstates API
// +kubebuilder:resource:path=nmstates,scope=Cluster
type NMstate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NMstateList contains a list of NMstate
type NMstateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NMstate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NMstate{}, &NMstateList{})
}
