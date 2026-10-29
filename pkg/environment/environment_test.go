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

package environment

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetEnvVarAsInt", func() {
	const key = "TEST_ENV_AS_INT"

	It("returns the default when unset", func() {
		value, err := GetEnvVarAsInt(key, 5)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(5))
	})

	DescribeTable("with a value set",
		func(set string, expected int, expectErr bool) {
			GinkgoT().Setenv(key, set)
			value, err := GetEnvVarAsInt(key, 5)
			if expectErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(expected))
		},
		Entry("accepts a positive integer", "3", 3, false),
		Entry("rejects zero", "0", 0, true),
		Entry("rejects a negative integer", "-1", 0, true),
		Entry("rejects a non-integer", "abc", 0, true),
		Entry("rejects an empty string", "", 0, true),
	)
})

var _ = Describe("GetEnvVarAsDuration", func() {
	const key = "TEST_ENV_AS_DURATION"

	It("returns the default when unset", func() {
		value, err := GetEnvVarAsDuration(key, 30*time.Second)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(30 * time.Second))
	})

	DescribeTable("with a value set",
		func(set string, expected time.Duration, expectErr bool) {
			GinkgoT().Setenv(key, set)
			value, err := GetEnvVarAsDuration(key, 30*time.Second)
			if expectErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(expected))
		},
		Entry("accepts a positive number of seconds", "10", 10*time.Second, false),
		Entry("rejects zero", "0", time.Duration(0), true),
		Entry("rejects a negative integer", "-5", time.Duration(0), true),
		Entry("rejects a non-integer", "1.5", time.Duration(0), true),
		Entry("rejects an empty string", "", time.Duration(0), true),
	)
})
