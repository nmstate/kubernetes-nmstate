package conditions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

const (
	failing     = nmstate.NodeNetworkConfigurationEnactmentConditionFailing
	available   = nmstate.NodeNetworkConfigurationEnactmentConditionAvailable
	progressing = nmstate.NodeNetworkConfigurationEnactmentConditionProgressing
	aborted     = nmstate.NodeNetworkConfigurationEnactmentConditionAborted
	t           = corev1.ConditionTrue
	f           = corev1.ConditionFalse
	u           = corev1.ConditionUnknown
)

type setter = func(*nmstate.ConditionList, string)

func enactments(enactments ...nmstatev1beta1.NodeNetworkConfigurationEnactment) nmstatev1beta1.NodeNetworkConfigurationEnactmentList {
	return nmstatev1beta1.NodeNetworkConfigurationEnactmentList{
		Items: append([]nmstatev1beta1.NodeNetworkConfigurationEnactment{}, enactments...),
	}
}

func enactment(policyGeneration int64, setters ...setter) nmstatev1beta1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{
		Status: nmstate.NodeNetworkConfigurationEnactmentStatus{
			PolicyGeneration: policyGeneration,
			Conditions:       nmstate.ConditionList{},
		},
	}
	for _, setter := range setters {
		setter(&enactment.Status.Conditions, "")
	}
	return enactment
}

var _ = Describe("Enactment condition counter", func() {
	type EnactmentCounterCase struct {
		enactmentsToCount nmstatev1beta1.NodeNetworkConfigurationEnactmentList
		policyGeneration  int64
		expectedCount     ConditionCount
	}
	DescribeTable("the enactments statuses", func(c EnactmentCounterCase) {
		obtainedCount := Count(c.enactmentsToCount, c.policyGeneration)
		Expect(obtainedCount).To(Equal(c.expectedCount))
	},
		Entry("e(), e()", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1),
				enactment(1),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 0, f: 0, u: 2},
				aborted:     CountByConditionStatus{t: 0, f: 0, u: 2},
			},
		}),
		Entry("e(Failed), e(Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetFailedToConfigure),
				enactment(1, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 1, f: 0, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				aborted:     CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(Success), e(Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetSuccess),
				enactment(1, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 1, f: 0, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				aborted:     CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(Progressing), e(Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetProgressing),
				enactment(1, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 2, f: 0, u: 0},
				aborted:     CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(Success), e(Success)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetSuccess),
				enactment(1, SetSuccess),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 2, f: 0, u: 0},
				failing:     CountByConditionStatus{t: 0, f: 2, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				aborted:     CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(Failed), e(Failed)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetFailedToConfigure),
				enactment(1, SetFailedToConfigure),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 2, f: 0, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				aborted:     CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(Failed), e(Aborted)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetFailedToConfigure),
				enactment(1, SetConfigurationAborted),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 1, f: 1, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				aborted:     CountByConditionStatus{t: 1, f: 1, u: 0},
			},
		}),
		Entry("p(2), e(1,Progressing), e(2,Progressing)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetProgressing),
				enactment(2, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 1, f: 0, u: 1},
				aborted:     CountByConditionStatus{t: 0, f: 1, u: 1},
			},
		}),
		Entry("p(2), e(1,Success), e(2,Success)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetSuccess),
				enactment(2, SetSuccess),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 1, f: 0, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				aborted:     CountByConditionStatus{t: 0, f: 1, u: 1},
			},
		}),
		Entry("p(2), e(1,Failed), e(2,Failed)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetFailedToConfigure),
				enactment(2, SetFailedToConfigure),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 1, f: 0, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				aborted:     CountByConditionStatus{t: 0, f: 1, u: 1},
			},
		}),
		Entry("p(2), e(1,Failed), e(2,Aborted)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetFailedToConfigure),
				enactment(2, SetConfigurationAborted),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				aborted:     CountByConditionStatus{t: 1, f: 0, u: 1},
			},
		}),
	)
})
