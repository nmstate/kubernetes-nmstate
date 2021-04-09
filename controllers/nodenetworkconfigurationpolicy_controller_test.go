package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type predicateCase struct {
		GenerationOld   int64
		GenerationNew   int64
		ReconcileCreate bool
		ReconcileUpdate bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
			oldNNCP := nmstatev1beta1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationOld,
				},
			}
			newNNCP := nmstatev1beta1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationNew,
				},
			}

			predicate := onCreateOrUpdateWithDifferentGeneration

			Expect(predicate.
				CreateFunc(event.CreateEvent{
					Object: &newNNCP,
				})).To(Equal(c.ReconcileCreate))
			Expect(predicate.
				UpdateFunc(event.UpdateEvent{
					ObjectOld: &oldNNCP,
					ObjectNew: &newNNCP,
				})).To(Equal(c.ReconcileUpdate))
			Expect(predicate.
				DeleteFunc(event.DeleteEvent{
					Object: &newNNCP,
				})).To(BeFalse())
		},
		Entry("generation remains the same",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   1,
				ReconcileCreate: true,
				ReconcileUpdate: false,
			}),
		Entry("generation is different",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileCreate: true,
				ReconcileUpdate: true,
			}),
	)
})
