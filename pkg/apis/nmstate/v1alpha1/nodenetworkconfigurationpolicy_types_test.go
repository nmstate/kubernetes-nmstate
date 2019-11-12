package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Policy Enactments list", func() {
	Context("is empty", func() {
		var originalEnactments, newEnactments EnactmentList

		BeforeEach(func() {
			originalEnactments = EnactmentList{}
		})

		It("should return nil when finding a condition", func() {
			foundCondition := newEnactments.FindCondition("foo_node", ConditionType("bar"))
			Expect(foundCondition).To(BeNil())
		})

		Context("and we set a new condition", func() {
			addedNodeName := "shruberry"
			addedConditionType := ConditionType("foo")
			addedConditionStatus := corev1.ConditionTrue
			addedConditionReason := ConditionReason("bar")
			addedConditionMessage := "baz"

			BeforeEach(func() {
				newEnactments.SetCondition(addedNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
			})

			It("should extend the list", func() {
				Expect(newEnactments).To(HaveLen(len(originalEnactments) + 1))
			})

			It("should add expected node entry to the list", func() {
				Expect(newEnactments[0].NodeName).To(Equal(addedNodeName))
			})

			It("should be able to find the added condition", func() {
				addedCondition := newEnactments.FindCondition(addedNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
				Expect(addedCondition.Type).To(Equal(addedConditionType))
				Expect(addedCondition.Status).To(Equal(addedConditionStatus))
			})
		})
	})

	Context("contains a single node info entry", func() {
		existingNodeName := "existing_shruberry"
		existingConditionType := ConditionType("existing_foo")
		existingConditionStatus := corev1.ConditionTrue
		existingConditionReason := ConditionReason("existing_bar")
		existingConditionMessage := "existing_baz"

		originalEnactments := EnactmentList{}
		originalEnactments.SetCondition(existingNodeName, existingConditionType, existingConditionStatus, existingConditionReason, existingConditionMessage)
		var newEnactments EnactmentList

		BeforeEach(func() {
			newEnactments = originalEnactments.DeepCopy()
		})

		Context("and we add a new condition to it", func() {
			addedConditionType := ConditionType("added_foo")
			addedConditionStatus := corev1.ConditionFalse
			addedConditionReason := ConditionReason("added_bar")
			addedConditionMessage := "added_baz"

			BeforeEach(func() {
				newEnactments.SetCondition(existingNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
			})

			It("should not add a new node entry to the list", func() {
				Expect(newEnactments).To(HaveLen(len(originalEnactments)))
			})

			It("should contain both the old and new conditions", func() {
				existingCondition := newEnactments.FindCondition(existingNodeName, existingConditionType)
				Expect(existingCondition).NotTo(BeNil())

				addedCondition := newEnactments.FindCondition(existingNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
			})
		})

		Context("and we change its condition", func() {
			updatedConditionStatus := corev1.ConditionFalse
			updatedConditionReason := ConditionReason("updated_bar")
			updatedConditionMessage := "updated_baz"

			BeforeEach(func() {
				newEnactments.SetCondition(existingNodeName, existingConditionType, updatedConditionStatus, updatedConditionReason, updatedConditionMessage)
			})

			It("should not add a new entry to the list", func() {
				Expect(newEnactments).To(HaveLen(len(originalEnactments)))
			})

			It("should not add a new condition to the existing entry", func() {
				Expect(newEnactments[0].Conditions).To(HaveLen(len(originalEnactments[0].Conditions)))
			})

			It("should be changed", func() {
				updatedCondition := newEnactments.FindCondition(existingNodeName, existingConditionType)
				Expect(updatedCondition.Status).To(Equal(updatedConditionStatus))
			})
		})

		Context("and we add a new one", func() {
			addedNodeName := "added_shruberry"
			addedConditionType := ConditionType("added_foo")
			addedConditionStatus := corev1.ConditionFalse
			addedConditionReason := ConditionReason("added_bar")
			addedConditionMessage := "added_baz"

			BeforeEach(func() {
				newEnactments.SetCondition(addedNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
			})

			It("should extend the list", func() {
				Expect(newEnactments).To(HaveLen(len(originalEnactments) + 1))
			})

			It("should contain both the old and the new conditions", func() {
				existingCondition := newEnactments.FindCondition(existingNodeName, existingConditionType)
				Expect(existingCondition).NotTo(BeNil())

				addedCondition := newEnactments.FindCondition(addedNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
			})
		})
	})
})
