package shared

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Conditions list", func() {
	Context("is empty", func() {
		It("should return nil when finding a condition", func() {
			condition := ConditionList{}.Find(NodeNetworkStateConditionFailing)
			Expect(condition).To(BeNil())
		})
	})

	Context("contains a single item", func() {
		originalConditions := ConditionList{
			Condition{
				Type:    NodeNetworkStateConditionFailing,
				Status:  corev1.ConditionUnknown,
				Reason:  ConditionReason("foo"),
				Message: "bar",
				LastHeartbeatTime: metav1.Time{
					Time: time.Unix(0, 0),
				},
				LastTransitionTime: metav1.Time{
					Time: time.Unix(0, 0),
				},
			},
		}
		var newConditions ConditionList

		BeforeEach(func() {
			newConditions = originalConditions.DeepCopy()
		})

		Context("and we add a new one", func() {
			addedConditionType := NodeNetworkStateConditionAvailable
			addedConditionStatus := corev1.ConditionTrue
			addedConditionReason := ConditionReason("foo")
			addedConditionMessage := "bar"

			BeforeEach(func() {
				newConditions.Set(addedConditionType, addedConditionStatus, addedConditionReason, addedConditionMessage)
			})

			It("should extend the list", func() {
				Expect(newConditions).To(HaveLen(len(originalConditions) + 1))
			})

			It("should set expected values to the added condition", func() {
				addedCondition := newConditions.Find(addedConditionType)
				Expect(addedCondition.Type).To(Equal(addedConditionType))
				Expect(addedCondition.Status).To(Equal(addedConditionStatus))
				Expect(addedCondition.Reason).To(Equal(addedConditionReason))
				Expect(addedCondition.Message).To(Equal(addedConditionMessage))
				Expect(addedCondition.LastTransitionTime.Time).To(BeTemporally("~", time.Now()))
				Expect(addedCondition.LastHeartbeatTime.Time).To(BeTemporally("==", addedCondition.LastTransitionTime.Time))
			})

			It("should not change the existing condition", func() {
				preexistingCondition := newConditions.Find(originalConditions[0].Type)
				Expect(preexistingCondition.Type).To(Equal(originalConditions[0].Type))
				Expect(preexistingCondition.Status).To(Equal(originalConditions[0].Status))
				Expect(preexistingCondition.Reason).To(Equal(originalConditions[0].Reason))
				Expect(preexistingCondition.Message).To(Equal(originalConditions[0].Message))
				Expect(preexistingCondition.LastTransitionTime.Time).To(BeTemporally("~", originalConditions[0].LastTransitionTime.Time))
				Expect(preexistingCondition.LastHeartbeatTime.Time).To(BeTemporally("==", originalConditions[0].LastHeartbeatTime.Time))
			})
		})

		Context("and we update it with the same values", func() {
			BeforeEach(func() {
				newConditions.Set(originalConditions[0].Type, originalConditions[0].Status, originalConditions[0].Reason, originalConditions[0].Message)
			})

			It("should not add a new condition", func() {
				Expect(newConditions).To(HaveLen(len(originalConditions)))
			})

			It("should change LastHearbeatTime and keep LastTransitionTime", func() {
				updatedCondition := newConditions.Find(originalConditions[0].Type)
				Expect(updatedCondition.LastHeartbeatTime.Time).To(BeTemporally(">", originalConditions[0].LastHeartbeatTime.Time))
				Expect(updatedCondition.LastTransitionTime.Time).To(BeTemporally("==", originalConditions[0].LastTransitionTime.Time))
			})
		})

		Context("and we update it with different values", func() {
			updatedConditionStatus := corev1.ConditionTrue
			updatedConditionReason := ConditionReason("bar")
			updatedConditionMessage := "foo"

			BeforeEach(func() {
				newConditions.Set(originalConditions[0].Type, updatedConditionStatus, updatedConditionReason, updatedConditionMessage)
			})

			It("should not add a new condition", func() {
				Expect(newConditions).To(HaveLen(len(originalConditions)))
			})

			It("should change values and update LastTransitionTime and LastHearbeatTime", func() {
				updatedCondition := newConditions.Find(originalConditions[0].Type)
				Expect(updatedCondition.Status).To(Equal(updatedConditionStatus))
				Expect(updatedCondition.Reason).To(Equal(updatedConditionReason))
				Expect(updatedCondition.Message).To(Equal(updatedConditionMessage))
				Expect(updatedCondition.LastTransitionTime.Time).To(BeTemporally(">", originalConditions[0].LastTransitionTime.Time))
				Expect(updatedCondition.LastHeartbeatTime.Time).To(BeTemporally(">", originalConditions[0].LastHeartbeatTime.Time))
			})
		})
	})
})
