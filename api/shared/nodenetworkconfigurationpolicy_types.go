package shared

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeNetworkConfigurationPolicySpec defines the desired state of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationPolicySpec struct {
	// NodeSelector is a selector which must be true for the policy to be applied to the node.
	// Selector which must match a node's labels for the policy to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:XPreserveUnknownFields
	// The desired configuration of the policy
	DesiredState State `json:"desiredState,omitempty"`

	// +kubebuilder:validation:XPreserveUnknownFields
	// When set to true, changes are applied to all nodes in parallel
	// +optional
	Parallel bool `json:"parallel,omitempty"`
}

// NodeNetworkConfigurationPolicyStatus defines the observed state of NodeNetworkConfigurationPolicy
type NodeNetworkConfigurationPolicyStatus struct {
	Conditions ConditionList `json:"conditions,omitempty" optional:"true"`

	// NodeRunningUpdate field is used for serializing cluster nodes configuration
	// +optional
	NodeRunningUpdate string `json:"nodeRunningUpdate,omitempty" optional:"true"`

	// NodeUpdateStart marks starting time of a node on a policy configuration
	// +optional
	NodeUpdateStart *metav1.Time `json:"nodeUpdateStart,omitempty" optional:"true"`
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
