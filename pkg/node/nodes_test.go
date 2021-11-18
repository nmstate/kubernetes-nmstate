package node

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("MaxUnavailable nodes", func() {
	type maxUnavailableCase struct {
		nmstateEnabledNodes          int
		maxUnavailable               intstr.IntOrString
		expectedScaledMaxUnavailable int
		expectedError                types.GomegaMatcher
	}
	DescribeTable("testing ScaledMaxUnavailableNodeCount",
		func(c maxUnavailableCase) {
			maxUnavailable, err := ScaledMaxUnavailableNodeCount(c.nmstateEnabledNodes, c.maxUnavailable)
			Expect(err).To(c.expectedError)
			Expect(maxUnavailable).To(Equal(c.expectedScaledMaxUnavailable))
		},
		Entry("Default maxUnavailable with odd number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString(DEFAULT_MAXUNAVAILABLE),
				expectedScaledMaxUnavailable: 3,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Default maxUnavailable with even number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          6,
				maxUnavailable:               intstr.FromString(DEFAULT_MAXUNAVAILABLE),
				expectedScaledMaxUnavailable: 3,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Absolute maxUnavailable with odd number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromInt(3),
				expectedScaledMaxUnavailable: 3,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Absolute maxUnavailable with even number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          6,
				maxUnavailable:               intstr.FromInt(3),
				expectedScaledMaxUnavailable: 3,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Wrong string value",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("asdf"),
				expectedScaledMaxUnavailable: 3,
				expectedError:                HaveOccurred(),
			}),
		Entry("Absolute value in string",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("42"),
				expectedScaledMaxUnavailable: 3,
				expectedError:                HaveOccurred(),
			}),
		Entry("Zero percent",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("0%"),
				expectedScaledMaxUnavailable: 1,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Zero value",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromInt(0),
				expectedScaledMaxUnavailable: 1,
				expectedError:                Not(HaveOccurred()),
			}))
})
