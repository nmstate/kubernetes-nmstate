package shared

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

// NodeNetworkConfigurationEnactmentStatus defines the observed state of NodeNetworkConfigurationEnactment
// +k8s:openapi-gen=true
type NodeNetworkConfigurationEnactmentStatus struct {
	// +kubebuilder:validation:XPreserveUnknownFields
	// The desired state rendered for the enactment's node using
	// the policy desiredState as template
	DesiredState State `json:"desiredState,omitempty"`

	// The generation from policy needed to check if an enactment
	// condition status belongs to the same policy version
	PolicyGeneration int64         `json:"policyGeneration,omitempty"`
	Conditions       ConditionList `json:"conditions,omitempty"`
}

const (
	EnactmentPolicyLabel                                                = "nmstate.io/policy"
	NodeNetworkConfigurationEnactmentConditionAvailable   ConditionType = "Available"
	NodeNetworkConfigurationEnactmentConditionFailing     ConditionType = "Failing"
	NodeNetworkConfigurationEnactmentConditionProgressing ConditionType = "Progressing"
	NodeNetworkConfigurationEnactmentConditionMatching    ConditionType = "Matching"
)

var NodeNetworkConfigurationEnactmentConditionTypes = [...]ConditionType{
	NodeNetworkConfigurationEnactmentConditionAvailable,
	NodeNetworkConfigurationEnactmentConditionFailing,
	NodeNetworkConfigurationEnactmentConditionProgressing,
	NodeNetworkConfigurationEnactmentConditionMatching,
}

const (
	NodeNetworkConfigurationEnactmentConditionFailedToConfigure                ConditionReason = "FailedToConfigure"
	NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured           ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationEnactmentConditionConfigurationProgressing         ConditionReason = "ConfigurationProgressing"
	NodeNetworkConfigurationEnactmentConditionNodeSelectorNotMatching          ConditionReason = "NodeSelectorNotMatching"
	NodeNetworkConfigurationEnactmentConditionNodeSelectorAllSelectorsMatching ConditionReason = "AllSelectorsMatching"
)

func EnactmentKey(node string, policy string) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("%s.%s", node, policy)}
}
