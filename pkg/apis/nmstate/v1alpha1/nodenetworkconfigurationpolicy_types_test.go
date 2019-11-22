package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Policy Enactments list", func() {
	Context("is empty", func() {
		var originalPolicy, newPolicy NodeNetworkConfigurationPolicy

		BeforeEach(func() {
			originalPolicy = NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy-1",
				},
			}
			newPolicy = originalPolicy
		})

		It("should return nil when finding a condition", func() {
			foundCondition := newPolicy.FindEnactmentCondition("foo_node", ConditionType("bar"))
			Expect(foundCondition).To(BeNil())
		})
		It("should return error when seting condition", func() {
			err := newPolicy.SetEnactmentCondition("arrakis", ConditionType("foo"), corev1.ConditionTrue, ConditionReason("bar"), "baz")
			Expect(err).To(HaveOccurred())
		})
		Context("and we set a new message and condition", func() {
			message := "Enactment matching"
			addedNodeName := "shruberry"
			addedConditionType := ConditionType("foo")
			addedConditionStatus := corev1.ConditionTrue
			addedConditionReason := ConditionReason("bar")
			addedConditionMessage := "baz"

			BeforeEach(func() {
				newPolicy.SetEnactmentMessage(addedNodeName, message)
				err := newPolicy.SetEnactmentCondition(addedNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should extend the list", func() {
				Expect(newPolicy.Status.Enactments).To(HaveLen(len(originalPolicy.Status.Enactments) + 1))
			})

			It("should add expected node entry to the list", func() {
				Expect(newPolicy.Status.Enactments[0].NodeName).To(Equal(addedNodeName))
			})
			It("should create NodeNetworkConfigurationEnactment", func() {
				nnce := newPolicy.Status.Enactments[0].Ref
				Expect(nnce.Name).To(Equal(addedNodeName + "-" + originalPolicy.Name))
			})
			It("should be able to find the added condition", func() {
				addedCondition := newPolicy.FindEnactmentCondition(addedNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
				Expect(addedCondition.Type).To(Equal(addedConditionType))
				Expect(addedCondition.Status).To(Equal(addedConditionStatus))
			})
		})
	})
	Context("contains a single node info entry", func() {
		message := "Enactment matching"
		existingNodeName := "existing_shruberry"
		existingConditionType := ConditionType("existing_foo")
		existingConditionStatus := corev1.ConditionTrue
		existingConditionReason := ConditionReason("existing_bar")
		existingConditionMessage := "existing_baz"

		originalPolicy := NodeNetworkConfigurationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy-1",
			},
		}
		var newPolicy NodeNetworkConfigurationPolicy

		BeforeEach(func() {
			originalPolicy.SetEnactmentMessage(existingNodeName, message)
			err := originalPolicy.SetEnactmentCondition(existingNodeName, existingConditionType, existingConditionStatus, existingConditionReason, existingConditionMessage)
			Expect(err).ToNot(HaveOccurred())
			newPolicy = originalPolicy
		})

		Context("and we add a new condition to it", func() {
			addedConditionType := ConditionType("added_foo")
			addedConditionStatus := corev1.ConditionFalse
			addedConditionReason := ConditionReason("added_bar")
			addedConditionMessage := "added_baz"

			BeforeEach(func() {
				newPolicy.SetEnactmentMessage(existingNodeName, message)
				err := newPolicy.SetEnactmentCondition(existingNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not add a new node entry to the list", func() {
				Expect(newPolicy.Status.Enactments).To(HaveLen(len(originalPolicy.Status.Enactments)))
			})

			It("should contain both the old and new conditions", func() {
				existingCondition := newPolicy.FindEnactmentCondition(existingNodeName, existingConditionType)
				Expect(existingCondition).NotTo(BeNil())

				addedCondition := newPolicy.FindEnactmentCondition(existingNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
			})
		})
		Context("and we change its condition", func() {
			updatedConditionStatus := corev1.ConditionFalse
			updatedConditionReason := ConditionReason("updated_bar")
			updatedConditionMessage := "updated_baz"

			BeforeEach(func() {
				newPolicy.SetEnactmentMessage(existingNodeName, message)
				err := newPolicy.SetEnactmentCondition(existingNodeName, existingConditionType, updatedConditionStatus, updatedConditionReason, updatedConditionMessage)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not add a new entry to the list", func() {
				Expect(newPolicy.Status.Enactments).To(HaveLen(len(originalPolicy.Status.Enactments)))
			})

			It("should not add a new condition to the existing entry", func() {
				Expect(newPolicy.Status.Enactments[0].Ref.Status.Conditions).To(HaveLen(len(originalPolicy.Status.Enactments[0].Ref.Status.Conditions)))
			})

			It("should be changed", func() {
				updatedCondition := newPolicy.FindEnactmentCondition(existingNodeName, existingConditionType)
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
				newPolicy.SetEnactmentMessage(addedNodeName, message)
				err := newPolicy.SetEnactmentCondition(addedNodeName, addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should extend the list", func() {
				Expect(newPolicy.Status.Enactments).To(HaveLen(len(originalPolicy.Status.Enactments) + 1))
			})

			It("should contain both the old and the new conditions", func() {
				existingCondition := newPolicy.FindEnactmentCondition(existingNodeName, existingConditionType)
				Expect(existingCondition).NotTo(BeNil())

				addedCondition := newPolicy.FindEnactmentCondition(addedNodeName, addedConditionType)
				Expect(addedCondition).NotTo(BeNil())
			})
		})
	})
})
