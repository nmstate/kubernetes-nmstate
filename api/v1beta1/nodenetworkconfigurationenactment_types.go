package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
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
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationenactments,shortName=nnce,scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type",description="Status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].reason",description="Reason"
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:storageversion

// NodeNetworkConfigurationEnactment is the Schema for the nodenetworkconfigurationenactments API
type NodeNetworkConfigurationEnactment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status shared.NodeNetworkConfigurationEnactmentStatus `json:"status,omitempty"`
}

func NewEnactment(node *corev1.Node, policy nmstatev1.NodeNetworkConfigurationPolicy) NodeNetworkConfigurationEnactment {
	enactment := NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name: shared.EnactmentKey(node.Name, policy.Name).Name,
			OwnerReferences: []metav1.OwnerReference{
				{Name: node.Name, Kind: "Node", APIVersion: "v1", UID: node.UID},
			},
			// Associate policy and node with the enactment using labels
			Labels: names.IncludeRelationshipLabels(map[string]string{shared.EnactmentPolicyLabel: policy.Name, shared.EnactmentNodeLabel: node.Name}),
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
