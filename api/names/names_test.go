package names

import (
	"os"
	"reflect"

	. "github.com/onsi/ginkgo"
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
			Expect(labels[COMPONENT_LABEL_KEY]).To(Equal(expectedComponentLabel))
			Expect(labels[PART_OF_LABEL_KEY]).To(Equal(expectedPartOfLabel))
			Expect(labels[VERSION_LABEL_KEY]).To(Equal(expectedVersionLabel))
			Expect(labels[MANAGED_BY_LABEL_KEY]).To(Equal(expectedManagedByLabel))
		})

		It("Should update labels with the right values", func() {
			labels := IncludeRelationshipLabels(map[string]string{"foo": "bar", VERSION_LABEL_KEY: "old_version"})

			Expect(labels["foo"]).To(Equal("bar"))
			Expect(labels[COMPONENT_LABEL_KEY]).To(Equal(expectedComponentLabel))
			Expect(labels[PART_OF_LABEL_KEY]).To(Equal(expectedPartOfLabel))
			Expect(labels[VERSION_LABEL_KEY]).To(Equal(expectedVersionLabel))
			Expect(labels[MANAGED_BY_LABEL_KEY]).To(Equal(expectedManagedByLabel))
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
			Expect(labels[COMPONENT_LABEL_KEY]).To(Equal(expectedComponentLabel))
			Expect(labels[PART_OF_LABEL_KEY]).To(Equal(expectedPartOfLabel))

			_, found := labels[VERSION_LABEL_KEY]
			Expect(found).To(Equal(false))

			_, found = labels[MANAGED_BY_LABEL_KEY]
			Expect(found).To(Equal(false))
		})
	})
})
