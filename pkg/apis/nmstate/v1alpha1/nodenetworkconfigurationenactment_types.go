package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationEnactmentList contains a list of NodeNetworkConfigurationEnactment
type NodeNetworkConfigurationEnactmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkConfigurationEnactment `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationEnactment is the Schema for the nodenetworkconfigurationenactments API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationenactments,shortName=nnce,scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Foo"
type NodeNetworkConfigurationEnactment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status NodeNetworkConfigurationEnactmentStatus `json:"status,omitempty"`
}

// NodeNetworkConfigurationEnactmentStatus defines the observed state of NodeNetworkConfigurationEnactment
// +k8s:openapi-gen=true
type NodeNetworkConfigurationEnactmentStatus struct {
	//TODO: Add the desired/rendered state field.
	Conditions ConditionList  `json:"conditions,omitempty" optional:"true"`
	Phase      EnactmentPhase `json:"phase,omitempty" optional:"true"`
}

type EnactmentPhase string

const (
	NodeNetworkConfigurationEnactmentConditionAvailable   ConditionType = "Available"
	NodeNetworkConfigurationEnactmentConditionFailing     ConditionType = "Failing"
	NodeNetworkConfigurationEnactmentConditionProgressing ConditionType = "Progressing"
)

const (
	NodeNetworkConfigurationEnactmentPhaseAvailable   EnactmentPhase = "Available"
	NodeNetworkConfigurationEnactmentPhaseFailing     EnactmentPhase = "Failing"
	NodeNetworkConfigurationEnactmentPhaseProgressing EnactmentPhase = "Progressing"
	NodeNetworkConfigurationEnactmentPhaseUnknown     EnactmentPhase = "Unknown"
)

const (
	NodeNetworkConfigurationEnactmentConditionFailedToConfigure        ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured   ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationEnactmentConditionConfigurationProgressing ConditionReason = "ConfigurationProgressing"
)

func init() {
	SchemeBuilder.Register(&NodeNetworkConfigurationEnactment{}, &NodeNetworkConfigurationEnactmentList{})
}
