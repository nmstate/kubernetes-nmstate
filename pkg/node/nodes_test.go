/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func beInvalidMaxUnavailableError() types.GomegaMatcher {
	return WithTransform(func(err error) bool {
		return errors.As(err, &InvalidMaxUnavailableError{})
	}, BeTrue())
}

var _ = Describe("MaxUnavailable nodes", func() {
	type maxUnavailableCase struct {
		nmstateEnabledNodes          int
		maxUnavailable               intstr.IntOrString
		expectedScaledMaxUnavailable int
		expectedError                types.GomegaMatcher
	}
	// Note: rejection of maxUnavailable values of 0 / "0%" is enforced by the CRD CEL
	// rule and covered by the e2e tests in test/e2e/handler/nnce_conditions_test.go.
	// As defense-in-depth ScaledMaxUnavailableNodeCount additionally returns an
	// InvalidMaxUnavailableError for any non-positive computed value (e.g. for policies
	// persisted before the CRD was upgraded), which the cases below assert.
	DescribeTable("testing ScaledMaxUnavailableNodeCount",
		func(c maxUnavailableCase) {
			maxUnavailable, err := ScaledMaxUnavailableNodeCount(c.nmstateEnabledNodes, c.maxUnavailable)
			Expect(err).To(c.expectedError)
			Expect(maxUnavailable).To(Equal(c.expectedScaledMaxUnavailable))
		},
		Entry("Default maxUnavailable with odd number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString(DefaultMaxunavailable),
				expectedScaledMaxUnavailable: 3,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Default maxUnavailable with even number of nodes",
			maxUnavailableCase{
				nmstateEnabledNodes:          6,
				maxUnavailable:               intstr.FromString(DefaultMaxunavailable),
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
		Entry("Wrong string value is terminally invalid",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("asdf"),
				expectedScaledMaxUnavailable: 3,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Bare integer string above int range is terminally invalid",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("99999999999999999999"),
				expectedScaledMaxUnavailable: 3,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Absolute value in string is treated as a node count",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("42"),
				expectedScaledMaxUnavailable: 42,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Zero percent",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("0%"),
				expectedScaledMaxUnavailable: 0,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Zero value",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromInt(0),
				expectedScaledMaxUnavailable: 0,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Zero-equivalent percentage that bypasses CEL validation",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("00%"),
				expectedScaledMaxUnavailable: 0,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Negative percentage that bypasses CEL validation",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("-100%"),
				expectedScaledMaxUnavailable: -5,
				expectedError:                beInvalidMaxUnavailableError(),
			}),
		Entry("Valid policy that matches no nodes is not reported invalid",
			maxUnavailableCase{
				nmstateEnabledNodes:          0,
				maxUnavailable:               intstr.FromString("50%"),
				expectedScaledMaxUnavailable: 0,
				expectedError:                Not(HaveOccurred()),
			}),
		Entry("Upper-bound percentage of 100% is valid",
			maxUnavailableCase{
				nmstateEnabledNodes:          5,
				maxUnavailable:               intstr.FromString("100%"),
				expectedScaledMaxUnavailable: 5,
				expectedError:                Not(HaveOccurred()),
			}))
})
