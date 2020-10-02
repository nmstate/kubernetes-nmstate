package nodenetworkconfigurationpolicy

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
			oldNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationOld,
			}

			newNodeNetworkConfigurationPolicyMeta := metav1.ObjectMeta{
				Generation: c.GenerationNew,
			}

			nodeNetworkConfigurationPolicy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}

			predicate := watchPredicate

			Expect(predicate.
				CreateFunc(event.CreateEvent{
					Meta:   &newNodeNetworkConfigurationPolicyMeta,
					Object: &nodeNetworkConfigurationPolicy,
				})).To(Equal(c.ReconcileCreate))
			Expect(predicate.
				UpdateFunc(event.UpdateEvent{
					MetaOld:   &oldNodeNetworkConfigurationPolicyMeta,
					ObjectOld: &nodeNetworkConfigurationPolicy,
					MetaNew:   &newNodeNetworkConfigurationPolicyMeta,
					ObjectNew: &nodeNetworkConfigurationPolicy,
				})).To(Equal(c.ReconcileUpdate))
			Expect(predicate.
				DeleteFunc(event.DeleteEvent{
					Meta:   &newNodeNetworkConfigurationPolicyMeta,
					Object: &nodeNetworkConfigurationPolicy,
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
