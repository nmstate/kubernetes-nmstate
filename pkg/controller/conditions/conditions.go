package conditions

import (
	"time"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetCondition(conditions nmstatev1.ConditionList, conditionType nmstatev1.ConditionType, status corev1.ConditionStatus, reason, message string) nmstatev1.ConditionList {
	condition := Condition(conditions, conditionType)
	now := metav1.Time{
		Time: time.Now(),
	}
	// If there isn't condition we want to change add new one
	if condition == nil {
		condition := nmstatev1.Condition{
			Type:               conditionType,
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastHeartbeatTime:  now,
			LastTransitionTime: now,
		}
		return append(conditions, condition)
	}

	// If there is different status, reason or message update it
	if condition.Status != status || condition.Reason != reason || condition.Message != message {
		condition.Status = status
		condition.Reason = reason
		condition.Message = message
		condition.LastTransitionTime = now
	}

	condition.LastHeartbeatTime = now

	return conditions
}

func Condition(conditions nmstatev1.ConditionList, conditionType nmstatev1.ConditionType) *nmstatev1.Condition {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
