package conditions

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

type CountByConditionStatus map[corev1.ConditionStatus]int

type ConditionCount map[nmstate.ConditionType]CountByConditionStatus

func Count(enactments nmstatev1beta1.NodeNetworkConfigurationEnactmentList, policyGeneration int64) ConditionCount {
	conditionCount := ConditionCount{}
	for _, conditionType := range nmstate.NodeNetworkConfigurationEnactmentConditionTypes {
		conditionCount[conditionType] = CountByConditionStatus{
			corev1.ConditionTrue:    0,
			corev1.ConditionFalse:   0,
			corev1.ConditionUnknown: 0,
		}
		for _, enactment := range enactments.Items {
			condition := enactment.Status.Conditions.Find(conditionType)
			// If there is a condition status and it's from the current policy update
			if condition != nil && enactment.Status.PolicyGeneration == policyGeneration {
				conditionCount[conditionType][condition.Status] += 1
			} else {
				conditionCount[conditionType][corev1.ConditionUnknown] += 1
			}
		}
	}
	return conditionCount
}

func (c ConditionCount) failed() CountByConditionStatus {
	return c[nmstate.NodeNetworkConfigurationEnactmentConditionFailing]
}
func (c ConditionCount) progressing() CountByConditionStatus {
	return c[nmstate.NodeNetworkConfigurationEnactmentConditionProgressing]
}
func (c ConditionCount) available() CountByConditionStatus {
	return c[nmstate.NodeNetworkConfigurationEnactmentConditionAvailable]
}
func (c ConditionCount) matching() CountByConditionStatus {
	return c[nmstate.NodeNetworkConfigurationEnactmentConditionMatching]
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
