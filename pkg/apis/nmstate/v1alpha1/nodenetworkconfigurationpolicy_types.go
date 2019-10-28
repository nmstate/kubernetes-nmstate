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
	Nodes NodeInfoList `json:"nodes,omitempty" optional:"true"`
}

// TODO: This is a temporary solution. This list will be replaced by a dedicated
// NodeNetworkConfigurationEnactment object.
type NodeInfoList []NodeInfo

type NodeInfo struct {
	Name       string        `json:"name"`
	Conditions ConditionList `json:"conditions,omitempty"`
}

func NewNodeInfo(nodeName string) NodeInfo {
	return NodeInfo{
		Name:       nodeName,
		Conditions: ConditionList{},
	}
}

func (list *NodeInfoList) SetCondition(nodeName string, conditionType ConditionType, status corev1.ConditionStatus, reason ConditionReason, message string) {
	nodeInfo := list.find(nodeName)

	if nodeInfo == nil {
		nodeInfo := NewNodeInfo(nodeName)
		nodeInfo.Conditions.Set(conditionType, status, reason, message)
		*list = append(*list, nodeInfo)
		return
	}

	nodeInfo.Conditions.Set(conditionType, status, reason, message)
}

func (list NodeInfoList) FindCondition(nodeName string, conditionType ConditionType) *Condition {
	nodeInfo := list.find(nodeName)
	if nodeInfo == nil {
		return nil
	}
	return nodeInfo.Conditions.Find(conditionType)
}

func (list NodeInfoList) find(nodeName string) *NodeInfo {
	for i, nodeInfo := range list {
		if nodeInfo.Name == nodeName {
			return &list[i]
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
