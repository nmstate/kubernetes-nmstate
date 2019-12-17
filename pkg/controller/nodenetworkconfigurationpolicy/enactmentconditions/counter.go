package enactmentconditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type CountByConditionType map[nmstatev1alpha1.ConditionType]int

func Count(enactments nmstatev1alpha1.NodeNetworkConfigurationEnactmentList) CountByConditionType {
	trueConditionsCount := CountByConditionType{}
	for _, enactment := range enactments.Items {
		for _, conditionType := range nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionTypes {
			condition := enactment.Status.Conditions.Find(conditionType)
			if condition != nil {
				if condition.Status == corev1.ConditionTrue {
					trueConditionsCount[conditionType] += 1
				}
			}
		}
	}
	return trueConditionsCount
}

func (c CountByConditionType) Failed() int {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing]
}
func (c CountByConditionType) Progressing() int {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing]
}
func (c CountByConditionType) Available() int {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable]
}

func (c CountByConditionType) String() string {
	return fmt.Sprintf("{failed: %d, progressing: %d, available: %d}", c.Failed(), c.Progressing(), c.Available())
}
