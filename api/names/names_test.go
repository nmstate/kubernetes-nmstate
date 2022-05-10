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

package names

import (
	"os"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IncludeRelationshipLabels", func() {
	Context("When env vars don't exist", func() {
		BeforeEach(func() {
			os.Unsetenv("COMPONENT")
			os.Unsetenv("PART_OF")
			os.Unsetenv("VERSION")
			os.Unsetenv("MANAGED_BY")
		})

		It("Should return empty map when input was nil map", func() {
			labels := IncludeRelationshipLabels(nil)
			Expect(labels).To(HaveLen(0))
		})

		It("Should return the same map when input was not nil", func() {
			labelsBase := map[string]string{"foo": "bar"}
			labels := IncludeRelationshipLabels(map[string]string{"foo": "bar"})
			Expect(reflect.DeepEqual(labelsBase, labels)).To(Equal(true))
		})
	})

	Context("When all the env vars are not empty", func() {
		const expectedComponentLabel = "component_unit_tests"
		const expectedPartOfLabel = "part_of_unit_tests"
		const expectedVersionLabel = "version_of_unit_tests"
		const expectedManagedByLabel = "managed_by_unit_tests"

		BeforeEach(func() {
			os.Setenv("COMPONENT", expectedComponentLabel)
			os.Setenv("PART_OF", expectedPartOfLabel)
			os.Setenv("VERSION", expectedVersionLabel)
			os.Setenv("MANAGED_BY", expectedManagedByLabel)
		})

		It("Should return labels with all the values", func() {
			labels := IncludeRelationshipLabels(map[string]string{"foo": "bar"})

			Expect(labels["foo"]).To(Equal("bar"))
			Expect(labels[ComponentLabelKey]).To(Equal(expectedComponentLabel))
			Expect(labels[PartOfLabelKey]).To(Equal(expectedPartOfLabel))
			Expect(labels[VersionLabelKey]).To(Equal(expectedVersionLabel))
			Expect(labels[ManagedByLabelKey]).To(Equal(expectedManagedByLabel))
		})

		It("Should update labels with the right values", func() {
			labels := IncludeRelationshipLabels(map[string]string{"foo": "bar", VersionLabelKey: "old_version"})

			Expect(labels["foo"]).To(Equal("bar"))
			Expect(labels[ComponentLabelKey]).To(Equal(expectedComponentLabel))
			Expect(labels[PartOfLabelKey]).To(Equal(expectedPartOfLabel))
			Expect(labels[VersionLabelKey]).To(Equal(expectedVersionLabel))
			Expect(labels[ManagedByLabelKey]).To(Equal(expectedManagedByLabel))
		})
	})

	Context("When some of the env vars are not empty", func() {
		const expectedComponentLabel = "component_unit_tests"
		const expectedPartOfLabel = "part_of_unit_tests"

		BeforeEach(func() {
			os.Setenv("COMPONENT", expectedComponentLabel)
			os.Setenv("PART_OF", expectedPartOfLabel)
			os.Setenv("VERSION", "")
			os.Unsetenv("MANAGED_BY")
		})

		It("Should return labels with the right values", func() {
			labels := IncludeRelationshipLabels(map[string]string{"foo": "bar"})
			Expect(labels["foo"]).To(Equal("bar"))
			Expect(labels[ComponentLabelKey]).To(Equal(expectedComponentLabel))
			Expect(labels[PartOfLabelKey]).To(Equal(expectedPartOfLabel))

			_, found := labels[VersionLabelKey]
			Expect(found).To(Equal(false))

			_, found = labels[ManagedByLabelKey]
			Expect(found).To(Equal(false))
		})
	})
})
