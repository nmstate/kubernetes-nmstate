package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationPolicyList contains a list of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationEnactmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkConfigurationEnactment `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationPolicy is the Schema for the nodenetworkconfigurationpolicies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationenactments,shortName=nncp,scope=Cluster
type NodeNetworkConfigurationEnactment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            NodeNetworkConfigurationEnactmentStatus `json:"status,omitempty"`
}

// NodeNetworkConfigurationPolicyStatus defines the observed state of NodeNetworkConfigurationPolicy
// +k8s:openapi-gen=true
type NodeNetworkConfigurationEnactmentStatus struct {
	Conditions ConditionList `json:"conditions,omitempty"`
}

func NewEnactment(nodeName string) NodeNetworkConfigurationEnactment {
	return NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}
}

func (enactments *NodeNetworkConfigurationEnactmentList) SetCondition(nodeName string, conditionType ConditionType, status corev1.ConditionStatus, reason ConditionReason, message string) {
	enactment := enactments.find(nodeName)

	if enactment == nil {
		enactment := NewEnactment(nodeName)
		enactment.Status.Conditions.Set(conditionType, status, reason, message)
		enactments.Items = append(enactments.Items, enactment)
		return
	}

	enactment.Status.Conditions.Set(conditionType, status, reason, message)
}

func (enactments NodeNetworkConfigurationEnactmentList) FindCondition(nodeName string, conditionType ConditionType) *Condition {
	enactment := enactments.find(nodeName)
	if enactment == nil {
		return nil
	}
	return enactment.Status.Conditions.Find(conditionType)
}

func (enactments NodeNetworkConfigurationEnactmentList) find(nodeName string) *NodeNetworkConfigurationEnactment {
	for i, enactment := range enactments.Items {
		if enactment.Name == nodeName {
			return &enactments.Items[i]
		}
	}
	return nil
}

const (
	NodeNetworkConfigurationEnactmentConditionAvailable   ConditionType = "Available"
	NodeNetworkConfigurationEnactmentConditionFailing     ConditionType = "Failing"
	NodeNetworkConfigurationEnactmentConditionProgressing ConditionType = "Progressing"
)

const (
	NodeNetworkConfigurationEnactmentConditionFailedToFindPolicy       ConditionReason = "FailedToFindPolicy"
	NodeNetworkConfigurationEnactmentConditionFailedToConfigure        ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured   ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationEnactmentConditionConfigurationProgressing ConditionReason = "ConfigurationProgressing"
)
