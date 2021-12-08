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
