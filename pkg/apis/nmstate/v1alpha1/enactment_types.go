package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

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
	NodeNetworkConfigurationEnactmentConditionMatching    ConditionType = "Matching"
	NodeNetworkConfigurationEnactmentConditionAvailable   ConditionType = "Available"
	NodeNetworkConfigurationEnactmentConditionFailing     ConditionType = "Failing"
	NodeNetworkConfigurationEnactmentConditionProgressing ConditionType = "Progressing"
)

const (
	NodeNetworkConfigurationEnactmentConditionReason                                   = "FailedToConfigure"
	NodeNetworkConfigurationEnactmentConditionSuccessfullyConfigured   ConditionReason = "SuccessfullyConfigured"
	NodeNetworkConfigurationEnactmentConditionConfigurationProgressing ConditionReason = "ConfigurationProgressing"
)
