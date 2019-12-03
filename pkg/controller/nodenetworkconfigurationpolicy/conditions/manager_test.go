package conditions

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type Enactment struct {
	Conditions nmstatev1alpha1.ConditionList
	Policy     string
}

func enactment(policy string, conditionsSetter func(*nmstatev1alpha1.ConditionList, string)) Enactment {
	conditions := nmstatev1alpha1.ConditionList{}
	conditionsSetter(&conditions, "")
	return Enactment{
		Policy:     policy,
		Conditions: conditions,
	}
}

func policyProgressing() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setPolicyProgressing(&conditions, "TODO")
	return conditions
}

func policySuccess() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setPolicySuccess(&conditions, "TODO")
	return conditions
}

func policyDegraded() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setPolicyFailedToConfigure(&conditions, "TODO")
	return conditions
}

func policyNotMatching() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setPolicyNotMatching(&conditions, "TODO")
	return conditions
}

func ignoreTimestamps(conditions nmstatev1alpha1.ConditionList) []GomegaMatcher {
	matchers := []GomegaMatcher{}
	for _, condition := range conditions {
		matchers = append(matchers, MatchFields(IgnoreExtras, Fields{
			"Type":    Equal(condition.Type),
			"Status":  Equal(condition.Status),
			"Reason":  Equal(condition.Reason),
			"Message": Equal(condition.Message),
		}))
	}
	return matchers
}

var _ = Describe("Conditions manager", func() {
	type ConditionsCase struct {
		Enactments       []Enactment
		NumberOfNodes    int
		PolicyConditions nmstatev1alpha1.ConditionList
	}
	DescribeTable("the policy overall condition",
		func(c ConditionsCase) {
			objs := []runtime.Object{}
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion,
				&nmstatev1alpha1.NodeNetworkConfigurationPolicy{},
				&nmstatev1alpha1.NodeNetworkConfigurationEnactment{},
				&nmstatev1alpha1.NodeNetworkConfigurationEnactmentList{},
			)
			policy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy1",
				},
			}
			for i, enactment := range c.Enactments {
				nnce := nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
				nodeName := fmt.Sprintf("node%d", i)
				nnce.Name = nmstatev1alpha1.EnactmentKey(nodeName, enactment.Policy).Name
				nnce.Status.Conditions = enactment.Conditions
				nnce.Labels = map[string]string{
					"policy": enactment.Policy,
				}
				objs = append(objs, &nnce)
			}
			for i := 1; i <= c.NumberOfNodes; i++ {
				node := corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("node%d", i),
					},
				}
				objs = append(objs, &node)
			}

			objs = append(objs, &policy)

			manager := Manager{
				client: fake.NewFakeClientWithScheme(s, objs...),
				policy: &policy,
			}
			err := manager.refreshPolicyConditions()
			Expect(err).ToNot(HaveOccurred())
			Expect(policy.Status.Conditions).To(ConsistOf(ignoreTimestamps(c.PolicyConditions)))
		},
		Entry("when all enactments are progressing then policy is progressing", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentProgressing),
				enactment("policy1", setEnactmentProgressing),
				enactment("policy1", setEnactmentProgressing),
			},
			NumberOfNodes:    3,
			PolicyConditions: policyProgressing(),
		}),
		Entry("when all enactments are success then policy is success", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentSuccess),
			},
			NumberOfNodes:    3,
			PolicyConditions: policySuccess(),
		}),
		Entry("when partial enactments are success then policy is progressing", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentSuccess),
			},
			NumberOfNodes:    4,
			PolicyConditions: policyProgressing(),
		}),
		Entry("when enactments are progressing/success then policy is progressing", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentProgressing),
				enactment("policy1", setEnactmentSuccess),
			},
			NumberOfNodes:    3,
			PolicyConditions: policyProgressing(),
		}),
		Entry("when enactments are failed/progressing/success then policy is degraded", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentSuccess),
				enactment("policy1", setEnactmentProgressing),
				enactment("policy1", setEnactmentFailedToConfigure),
				enactment("policy1", setEnactmentSuccess),
			},
			NumberOfNodes:    4,
			PolicyConditions: policyDegraded(),
		}),
		Entry("when neither of enactments are matching then policy is neither at degraded/progressing/success ", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentNodeSelectorNotMatching),
				enactment("policy1", setEnactmentNodeSelectorNotMatching),
				enactment("policy1", setEnactmentNodeSelectorNotMatching),
			},
			NumberOfNodes:    3,
			PolicyConditions: policyNotMatching(),
		}),
		Entry("when some enactments are from different profile then policy conditions are not affected by them", ConditionsCase{
			Enactments: []Enactment{
				enactment("policy1", setEnactmentSuccess),
				enactment("policy2", setEnactmentProgressing),
				enactment("policy2", setEnactmentProgressing),
			},
			NumberOfNodes:    3,
			PolicyConditions: policySuccess(),
		}),
	)
})
