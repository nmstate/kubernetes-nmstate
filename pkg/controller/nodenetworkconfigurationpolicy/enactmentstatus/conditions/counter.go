package conditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type CountByConditionStatus map[corev1.ConditionStatus]int

type ConditionCount map[nmstatev1alpha1.ConditionType]CountByConditionStatus

func Count(enactments nmstatev1alpha1.NodeNetworkConfigurationEnactmentList) ConditionCount {
	conditionCount := ConditionCount{}
	for _, conditionType := range nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionTypes {
		conditionCount[conditionType] = CountByConditionStatus{
			corev1.ConditionTrue:    0,
			corev1.ConditionFalse:   0,
			corev1.ConditionUnknown: 0,
		}
		for _, enactment := range enactments.Items {
			condition := enactment.Status.Conditions.Find(conditionType)
			if condition != nil {
				conditionCount[conditionType][condition.Status] += 1
			} else {
				conditionCount[conditionType][corev1.ConditionUnknown] += 1
			}
		}
	}
	return conditionCount
}

func (c ConditionCount) failed() CountByConditionStatus {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing]
}
func (c ConditionCount) progressing() CountByConditionStatus {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing]
}
func (c ConditionCount) available() CountByConditionStatus {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable]
}
func (c ConditionCount) matching() CountByConditionStatus {
	return c[nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching]
}

func (c CountByConditionStatus) true() int {
	return c[corev1.ConditionTrue]
}

func (c CountByConditionStatus) false() int {
	return c[corev1.ConditionFalse]
}

func (c CountByConditionStatus) unknown() int {
	return c[corev1.ConditionUnknown]
}

func (c ConditionCount) Failed() int {
	return c.failed().true()
}
func (c ConditionCount) NotFailed() int {
	return c.failed().false()
}
func (c ConditionCount) Progressing() int {
	return c.progressing().true()
}
func (c ConditionCount) NotProgressing() int {
	return c.progressing().false()
}
func (c ConditionCount) Available() int {
	return c.available().true()
}
func (c ConditionCount) NotAvailable() int {
	return c.available().false()
}
func (c ConditionCount) Matching() int {
	return c.matching().true()
}
func (c ConditionCount) NotMatching() int {
	return c.matching().false()
}

func (c ConditionCount) String() string {
	return fmt.Sprintf("{failed: %s, progressing: %s, available: %s, matching: %s}", c.failed(), c.progressing(), c.available(), c.matching())
}

func (c CountByConditionStatus) String() string {
	return fmt.Sprintf("{true: %d, false: %d, unknown: %d}", c.true(), c.false(), c.unknown())
}
