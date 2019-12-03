package conditions

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func enactmentProgressing() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setEnactmentProgressing(&conditions, "")
	return conditions
}

func policyProgressing() nmstatev1alpha1.ConditionList {
	conditions := nmstatev1alpha1.ConditionList{}
	setPolicyProgressing(&conditions, "")
	return conditions
}

var _ = Describe("Conditions manager", func() {
	type ConditionsCase struct {
		EnactmentsConditions []nmstatev1alpha1.ConditionList
		PolicyConditions     nmstatev1alpha1.ConditionList
	}
	DescribeTable("the policy overall condition",
		func(c ConditionsCase) {
			objs := []runtime.Object{}
			s := scheme.Scheme
			for i, enactmentConditions := range c.EnactmentsConditions {
				enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
				nodeName := fmt.Sprintf("node%d", i)
				enactment.Name = nmstatev1alpha1.EnactmentKey(nodeName, "policy-foo").Name
				enactment.Status.Conditions = enactmentConditions
				objs = append(objs, &enactment)
				s.AddKnownTypes(nmstatev1alpha1.SchemeGroupVersion, &enactment)
			}

			manager := Manager{
				client: fake.NewFakeClientWithScheme(s, objs...),
				policy: &nmstatev1alpha1.NodeNetworkConfigurationPolicy{},
			}
			manager.refreshPolicyConditions()

			Expect(manager.policy.Status.Conditions).To(ConsistOf(c.PolicyConditions))
		},
		Entry("All enactments at progressing state", ConditionsCase{
			EnactmentsConditions: []nmstatev1alpha1.ConditionList{
				enactmentProgressing(),
				enactmentProgressing(),
			},
			PolicyConditions: policyProgressing(),
		}),
	)
})
