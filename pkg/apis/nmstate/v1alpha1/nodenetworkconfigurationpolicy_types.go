package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationPolicyList contains a list of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeNetworkConfigurationPolicy `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationPolicy is the Schema for the nodenetworkconfigurationpolicies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationpolicies,shortName=nncp
type NodeNetworkConfigurationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeNetworkConfigurationPolicySpec   `json:"spec,omitempty"`
	Status NodeNetworkConfigurationPolicyStatus `json:"status,omitempty"`
}

// NodeNetworkConfigurationPolicySpec defines the desired state of NodeNetworkConfigurationPolicy
// +k8s:openapi-gen=true
type NodeNetworkConfigurationPolicySpec struct {
	// NodeSelector is a selector which must be true for the policy to be applied to the node.
	// Selector which must match a node's labels for the policy to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// The desired configuration of the policy
	DesiredState State `json:"desiredState,omitempty"`
}

// NodeNetworkConfigurationPolicyStatus defines the observed state of NodeNetworkConfigurationPolicy
// +k8s:openapi-gen=true
type NodeNetworkConfigurationPolicyStatus struct {
	Enactments EnactmentList `json:"enactments,omitempty" optional:"true"`
}

// TODO: This is a temporary solution. This list will be replaced by a dedicated
// NodeNetworkConfigurationEnactment object.
type EnactmentList []Enactment

type Enactment struct {
	NodeName   string        `json:"nodeName"`
	Conditions ConditionList `json:"conditions,omitempty"`
}

func NewEnactment(nodeName string) Enactment {
	return Enactment{
		NodeName:   nodeName,
		Conditions: ConditionList{},
	}
}

func (enactments *EnactmentList) SetCondition(nodeName string, conditionType ConditionType, status corev1.ConditionStatus, reason ConditionReason, message string) {
	enactment := enactments.find(nodeName)

	if enactment == nil {
		enactment := NewEnactment(nodeName)
		enactment.Conditions.Set(conditionType, status, reason, message)
		*enactments = append(*enactments, enactment)
		return
	}

	enactment.Conditions.Set(conditionType, status, reason, message)
}

func (enactments EnactmentList) FindCondition(nodeName string, conditionType ConditionType) *Condition {
	enactment := enactments.find(nodeName)
	if enactment == nil {
		return nil
	}
	return enactment.Conditions.Find(conditionType)
}

func (enactments EnactmentList) find(nodeName string) *Enactment {
	for i, enactment := range enactments {
		if enactment.NodeName == nodeName {
			return &enactments[i]
		}
	}
	return nil
}

const (
	NodeNetworkConfigurationPolicyConditionAvailable ConditionType = "Available"
	NodeNetworkConfigurationPolicyConditionFailing   ConditionType = "Failing"
)

const (
	NodeNetworkConfigurationPolicyConditionFailedToConfigure      ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured ConditionReason = "SuccessfullyConfigured"
)

func init() {
	SchemeBuilder.Register(&NodeNetworkConfigurationPolicy{}, &NodeNetworkConfigurationPolicyList{})
}
