package conditions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const (
	failing     = nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing
	available   = nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable
	progressing = nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing
	matching    = nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionMatching
	t           = corev1.ConditionTrue
	f           = corev1.ConditionFalse
	u           = corev1.ConditionUnknown
)

type setter = func(*nmstatev1alpha1.ConditionList, string)

func enactments(enactments ...nmstatev1alpha1.NodeNetworkConfigurationEnactment) nmstatev1alpha1.NodeNetworkConfigurationEnactmentList {
	return nmstatev1alpha1.NodeNetworkConfigurationEnactmentList{
		Items: append([]nmstatev1alpha1.NodeNetworkConfigurationEnactment{}, enactments...),
	}
}

func enactment(setters ...setter) nmstatev1alpha1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{
		Status: nmstatev1alpha1.NodeNetworkConfigurationEnactmentStatus{
			Conditions: nmstatev1alpha1.ConditionList{},
		},
	}
	for _, setter := range setters {
		setter(&enactment.Status.Conditions, "")
	}
	return enactment
}

var _ = Describe("Enactment condition counter", func() {
	type EnactmentCounterCase struct {
		enactmentsToCount nmstatev1alpha1.NodeNetworkConfigurationEnactmentList
		expectedCount     ConditionCount
	}
	DescribeTable("the enactments statuses", func(c EnactmentCounterCase) {
		obtainedCount := Count(c.enactmentsToCount)
		Expect(obtainedCount).To(Equal(c.expectedCount))
		//TODO: Do we also check getters ? available().true(), etc...
	},
		Entry("e(), e()", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(),
				enactment(),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 0, f: 0, u: 2},
				matching:    CountByConditionStatus{t: 0, f: 0, u: 2},
			},
		}),
		Entry("e(NotMatching), e(NotMatching)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetNodeSelectorNotMatching),
				enactment(SetNodeSelectorNotMatching),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 0, f: 2, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 0, f: 2, u: 0},
			},
		}),
		Entry("e(NotMatching), e(Matching, Progressing)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetNodeSelectorNotMatching),
				enactment(SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 1, f: 1, u: 0},
			},
		}),
		Entry("e(Matching, Failed), e(Matching, Progressing)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetMatching, SetFailedToConfigure),
				enactment(SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 1, u: 1},
				failing:     CountByConditionStatus{t: 1, f: 0, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Success), e(Matching, Progressing)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetMatching, SetSuccess),
				enactment(SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 1, f: 0, u: 1},
				failing:     CountByConditionStatus{t: 0, f: 1, u: 1},
				progressing: CountByConditionStatus{t: 1, f: 1, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Progressing), e(Matching, Progressing)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetMatching, SetProgressing),
				enactment(SetMatching, SetProgressing),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 0, u: 2},
				failing:     CountByConditionStatus{t: 0, f: 0, u: 2},
				progressing: CountByConditionStatus{t: 2, f: 0, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Success), e(Matching, Success)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetMatching, SetSuccess),
				enactment(SetMatching, SetSuccess),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 2, f: 0, u: 0},
				failing:     CountByConditionStatus{t: 0, f: 2, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
		Entry("e(Matching, Failed), e(Matching, Failed)", EnactmentCounterCase{
			enactmentsToCount: enactments(
				enactment(SetMatching, SetFailedToConfigure),
				enactment(SetMatching, SetFailedToConfigure),
			),
			expectedCount: ConditionCount{
				available:   CountByConditionStatus{t: 0, f: 2, u: 0},
				failing:     CountByConditionStatus{t: 2, f: 0, u: 0},
				progressing: CountByConditionStatus{t: 0, f: 2, u: 0},
				matching:    CountByConditionStatus{t: 2, f: 0, u: 0},
			},
		}),
	)
})
