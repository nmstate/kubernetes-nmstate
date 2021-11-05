package shared

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NodeNetworkConfigurationPolicySpec defines the desired state of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationPolicySpec struct {
	// NodeSelector is a selector which must be true for the policy to be applied to the node.
	// Selector which must match a node's labels for the policy to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Capture contains expressions with an associated name than can be referenced
	// at the DesiredState.
	Capture map[string]string `json:"capture,omitempty"`

	// +kubebuilder:validation:XPreserveUnknownFields
	// The desired configuration of the policy
	DesiredState State `json:"desiredState,omitempty"`

	// MaxUnavailable specifies percentage or number
	// of machines that can be updating at a time. Default is "50%".
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// NodeNetworkConfigurationPolicyStatus defines the observed state of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationPolicyStatus struct {
	Conditions ConditionList `json:"conditions,omitempty" optional:"true"`

	// UnavailableNodeCount represents the total number of potentially unavailable nodes that are
	// processing a NodeNetworkConfigurationPolicy
	// +optional
	UnavailableNodeCount int `json:"unavailableNodeCount,omitempty" optional:"true"`
	// LastUnavailableNodeCountUpdate is time of the last UnavailableNodeCount update
	// +optional
	LastUnavailableNodeCountUpdate *metav1.Time `json:"lastUnavailableNodeCountUpdate,omitempty" optional:"true"`
}

const (
	NodeNetworkConfigurationPolicyConditionAvailable ConditionType = "Available"
	NodeNetworkConfigurationPolicyConditionDegraded  ConditionType = "Degraded"
)

var NodeNetworkConfigurationPolicyConditionTypes = [...]ConditionType{
	NodeNetworkConfigurationPolicyConditionAvailable,
	NodeNetworkConfigurationPolicyConditionDegraded,
}

const (
	NodeNetworkConfigurationPolicyConditionFailedToConfigure           ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured      ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationPolicyConditionConfigurationProgressing    ConditionReason = "ConfigurationProgressing"
	NodeNetworkConfigurationPolicyConditionConfigurationNoMatchingNode ConditionReason = "NoMatchingNode"
)
