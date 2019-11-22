package v1alpha1

import (
	"fmt"

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
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeNetworkConfigurationPolicy is the Schema for the nodenetworkconfigurationpolicies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodenetworkconfigurationpolicies,shortName=nncp,scope=Cluster
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
	Enactments PolicyEnactmentList `json:"enactments,omitempty" optional:"true"`
}

// +k8s:openapi-gen=true
type PolicyEnactmentList []PolicyEnactment

// +k8s:openapi-gen=true
type PolicyEnactment struct {
	NodeName string `json:"nodeName,omitempty"`
	Message  string `json:"message,omitempty"`
	//TODO: Change this type to proper CDR lnk
	Ref *NodeNetworkConfigurationEnactment
}

const (
	NodeNetworkConfigurationPolicyConditionAvailable   ConditionType = "Available"
	NodeNetworkConfigurationPolicyConditionFailing     ConditionType = "Failing"
	NodeNetworkConfigurationPolicyConditionProgressing ConditionType = "Progressing"
)

const (
	NodeNetworkConfigurationPolicyConditionFailedToConfigure        ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured   ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationPolicyConditionConfigurationProgressing ConditionReason = "ConfigurationProgressing"
)

func (policy *NodeNetworkConfigurationPolicy) SetEnactmentMessage(nodeName string, message string) {
	enactment := policy.findEnactment(nodeName)
	if enactment == nil {
		policy.Status.Enactments = append(policy.Status.Enactments, PolicyEnactment{
			NodeName: nodeName,
			Message:  message,
		})
	} else {
		enactment.Message = message
	}
}

func (policy *NodeNetworkConfigurationPolicy) SetEnactmentCondition(nodeName string, conditionType ConditionType, status corev1.ConditionStatus, reason ConditionReason, message string) error {
	enactment := policy.findEnactment(nodeName)

	if enactment == nil {
		return fmt.Errorf("Enactment should be already there")
	}
	if enactment.Ref == nil {
		//TODO: Create the CR with client
		enactment.Ref = &NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName + "-" + policy.Name,
			},
		}
	}
	//TODO: Update the status with client
	enactment.Ref.Status.Conditions.Set(conditionType, status, reason, message)

	return nil
}

func (policy *NodeNetworkConfigurationPolicy) FindEnactmentCondition(nodeName string, conditionType ConditionType) *Condition {
	enactment := policy.findEnactment(nodeName)
	if enactment == nil {
		return nil
	}
	if enactment.Ref == nil {
		return nil
	}
	return enactment.Ref.Status.Conditions.Find(conditionType)
}

func (policy *NodeNetworkConfigurationPolicy) findEnactment(nodeName string) *PolicyEnactment {
	for i, enactment := range policy.Status.Enactments {
		if enactment.NodeName == nodeName {
			return &policy.Status.Enactments[i]
		}
	}
	return nil
}
func init() {
	SchemeBuilder.Register(&NodeNetworkConfigurationPolicy{}, &NodeNetworkConfigurationPolicyList{})
}
