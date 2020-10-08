package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

// +kubebuilder:object:root=true

// NodeNetworkConfigurationEnactmentList contains a list of NodeNetworkConfigurationEnactment
type NodeNetworkConfigurationEnactmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkConfigurationEnactment `json:"items"`
}

// +genclient
// +kubebuilder:object:root=true

// NodeNetworkConfigurationEnactment is the Schema for the nodenetworkconfigurationenactments API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationenactments,shortName=nnce,scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].reason",description="Status"
type NodeNetworkConfigurationEnactment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status shared.NodeNetworkConfigurationEnactmentStatus `json:"status,omitempty"`
}

func NewEnactment(nodeName string, policy NodeNetworkConfigurationPolicy) NodeNetworkConfigurationEnactment {
	enactment := NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name: shared.EnactmentKey(nodeName, policy.Name).Name,
			OwnerReferences: []metav1.OwnerReference{
				{Name: policy.Name, Kind: policy.TypeMeta.Kind, APIVersion: policy.TypeMeta.APIVersion, UID: policy.UID},
			},
			// Associate policy with the enactment using labels
			Labels: map[string]string{
				shared.EnactmentPolicyLabel: policy.Name,
			},
		},
		Status: shared.NodeNetworkConfigurationEnactmentStatus{
			DesiredState: shared.NewState(""),
			Conditions:   shared.ConditionList{},
		},
	}

	for _, conditionType := range shared.NodeNetworkConfigurationEnactmentConditionTypes {
		enactment.Status.Conditions.Set(conditionType, corev1.ConditionUnknown, "", "")
	}
	return enactment
}

func init() {
	SchemeBuilder.Register(&NodeNetworkConfigurationEnactment{}, &NodeNetworkConfigurationEnactmentList{})
}
