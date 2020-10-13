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
	matching    = nmstate.NodeNetworkConfigurationEnactmentConditionMatching
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
				matching:    CountByConditionStatus{t: 0, f: 0, u: 2},
			},
		}),
		Entry("e(NotMatching), e(NotMatching)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetNodeSelectorNotMatching),
				enactment(1, SetNodeSelectorNotMatching),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 0, f: 2, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(NotMatching), e(Matching, Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetNodeSelectorNotMatching),
				enactment(1, SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 1, f: 1, u: 0},
			},
		}),
		Entry("e(Matching, Failed), e(Matching, Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetFailedToConfigure),
				enactment(1, SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 1, f: 0, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Success), e(Matching, Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetSuccess),
				enactment(1, SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 1, f: 0, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Progressing), e(Matching, Progressing)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetProgressing),
				enactment(1, SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 2, f: 0, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Success), e(Matching, Success)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetSuccess),
				enactment(1, SetMatching, SetSuccess),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 2, f: 0, u: 0},
				failing:     CountByConditionStatus{t: 0, f: 2, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Failed), e(Matching, Failed)", EnactmentCounterCase{
			policyGeneration: 1,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetFailedToConfigure),
				enactment(1, SetMatching, SetFailedToConfigure),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 2, f: 0, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("p(2), e(1,NotMatching), e(2,NotMatching)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetNodeSelectorNotMatching),
				enactment(2, SetNodeSelectorNotMatching),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				matching:    CountByConditionStatus{t: 0, f: 1, u: 1},
			},
		}),
		Entry("p(2), e(1,Matching), e(2,Matching)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetMatching),
				enactment(2, SetMatching),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 0, f: 0, u: 2},
				matching:    CountByConditionStatus{t: 1, f: 0, u: 1},
			},
		}),
		Entry("p(2), e(1,Matching,Progressing), e(2,Matching,Progressing)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetProgressing),
				enactment(2, SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 1, f: 0, u: 1},
				matching:    CountByConditionStatus{t: 1, f: 0, u: 1},
			},
		}),
		Entry("p(2), e(1,Matching,Success), e(2,Matching,Success)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetSuccess),
				enactment(2, SetMatching, SetSuccess),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 1, f: 0, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				matching:    CountByConditionStatus{t: 1, f: 0, u: 1},
			},
		}),
		Entry("p(2), e(1,Matching,Failed), e(2,Matching,Failed)", EnactmentCounterCase{
			policyGeneration: 2,
			enactmentsToCount: enactments(
				enactment(1, SetMatching, SetFailedToConfigure),
				enactment(2, SetMatching, SetFailedToConfigure),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 1, f: 0, u: 1},
				progressing: CountByConditionStatus{t: 0, f: 1, u: 1},
				matching:    CountByConditionStatus{t: 1, f: 0, u: 1},
			},
		}),
	)
})
