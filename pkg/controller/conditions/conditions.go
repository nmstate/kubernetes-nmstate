package conditions

import (
	"time"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetCondition(instance *nmstatev1.NodeNetworkState, conditionType nmstatev1.NodeNetworkStateConditionType, status corev1.ConditionStatus, reason, message string) {
	condition := GetCondition(instance, conditionType)
	now := metav1.Time{
		Time: time.Now(),
	}
	// If there isn't condition we want to change add new one
	if condition == nil {
		condition := nmstatev1.NodeNetworkStateCondition{
			Type:               conditionType,
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastHeartbeatTime:  now,
			LastTransitionTime: now,
		}
		instance.Status.Conditions = append(instance.Status.Conditions, condition)
		return
	}

	// If there is different status, reason or message update it
	if condition.Status != status || condition.Reason != reason || condition.Message != message {
		condition.Status = status
		condition.Reason = reason
		condition.Message = message
		condition.LastTransitionTime = now
	}

	condition.LastHeartbeatTime = now
}

func GetCondition(instance *nmstatev1.NodeNetworkState, conditionType nmstatev1.NodeNetworkStateConditionType) *nmstatev1.NodeNetworkStateCondition {
	for i, condition := range instance.Status.Conditions {
		if condition.Type == conditionType {
			return &instance.Status.Conditions[i]
		}
	}
	return nil
}
