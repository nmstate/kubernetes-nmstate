package policyconditions

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	"github.com/nmstate/kubernetes-nmstate/pkg/controller/nodenetworkconfigurationpolicy/enactmentconditions"
)

func e(node string, policy string, conditionsSetter func(*nmstatev1alpha1.ConditionList, string)) nmstatev1alpha1.NodeNetworkConfigurationEnactment {
	conditions := nmstatev1alpha1.ConditionList{}
	conditionsSetter(&conditions, "")
	return nmstatev1alpha1.NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				nmstatev1alpha1.EnactmentPolicyLabel: policy,
			},
			Name: nmstatev1alpha1.EnactmentKey(node, policy).Name,
		},
		Status: nmstatev1alpha1.NodeNetworkConfigurationEnactmentStatus{
			Conditions: conditions,
		},
	}
}

func p(conditionsSetter func(*nmstatev1alpha1.ConditionList, string), message string, nodeSelector map[string]string) nmstatev1alpha1.NodeNetworkConfigurationPolicy {
	conditions := nmstatev1alpha1.ConditionList{}
	conditionsSetter(&conditions, message)
	return nmstatev1alpha1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "policy1",
		},
		Spec: nmstatev1alpha1.NodeNetworkConfigurationPolicySpec{
			NodeSelector: nodeSelector,
		},
		Status: nmstatev1alpha1.NodeNetworkConfigurationPolicyStatus{
			Conditions: conditions,
		},
	}
}

func cleanTimestamps(conditions nmstatev1alpha1.ConditionList) nmstatev1alpha1.ConditionList {
	dummyTime := metav1.Time{Time: time.Unix(0, 0)}
	for i, _ := range conditions {
		conditions[i].LastHeartbeatTime = dummyTime
		conditions[i].LastTransitionTime = dummyTime
	}
	return conditions
}

func allNodes() map[string]string {
	return map[string]string{}
}

func forNode(node string) map[string]string {
	return map[string]string{
		"kubernetes.io/hostname": node,
	}
}

var _ = Describe("Policy Conditions", func() {
	type ConditionsCase struct {
		Enactments    []nmstatev1alpha1.NodeNetworkConfigurationEnactment
		NumberOfNodes int
		Policy        nmstatev1alpha1.NodeNetworkConfigurationPolicy
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

			for i, _ := range c.Enactments {
				// We cannot use the memory from the element
				// returned by range, since it's has the same
				// memory address it will be added multiple time
				// we duplicated values
				objs = append(objs, &c.Enactments[i])
			}

			for i := 1; i <= c.NumberOfNodes; i++ {
				nodeName := fmt.Sprintf("node%d", i)
				node := corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: nodeName,
					},
				}
				node.Labels = map[string]string{
					"kubernetes.io/hostname": nodeName,
				}
				objs = append(objs, &node)
			}

			updatedPolicy := c.Policy.DeepCopy()
			updatedPolicy.Status.Conditions = nmstatev1alpha1.ConditionList{}

			objs = append(objs, updatedPolicy)

			client := fake.NewFakeClientWithScheme(s, objs...)
			err := Update(client, updatedPolicy)
			Expect(err).ToNot(HaveOccurred())
			Expect(cleanTimestamps(updatedPolicy.Status.Conditions)).To(ConsistOf(cleanTimestamps(c.Policy.Status.Conditions)))
		},
		Entry("when all enactments are progressing then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetProgressing),
				e("node2", "policy1", enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetProgressing),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicyProgressing, "Policy is progresssing at 3 nodes: {failed: 0, progressing: 3, available: 0}", allNodes()),
		}),
		Entry("when all enactments are success then policy is success", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetSuccess),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicySuccess, "3/3 nodes successfully configured", allNodes()),
		}),
		Entry("when partial enactments are success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetSuccess),
			},
			NumberOfNodes: 4,
			Policy:        p(setPolicyProgressing, "Policy is progresssing at 4 nodes: {failed: 0, progressing: 0, available: 3}", allNodes()),
		}),
		Entry("when enactments are progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetSuccess),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicyProgressing, "Policy is progresssing at 3 nodes: {failed: 0, progressing: 1, available: 2}", allNodes()),
		}),
		Entry("when enactments are failed/progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetFailedToConfigure),
				e("node4", "policy1", enactmentconditions.SetSuccess),
			},
			NumberOfNodes: 4,
			Policy:        p(setPolicyProgressing, "Policy is progresssing at 4 nodes: {failed: 1, progressing: 1, available: 2}", allNodes()),
		}),
		Entry("when all the enactments are at failing or success policy is degraded", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetSuccess),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicyFailedToConfigure, "2/3 nodes failed to configure", allNodes()),
		}),
		Entry("when all the enactments are at failing policy is degraded", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetFailedToConfigure),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicyFailedToConfigure, "3/3 nodes failed to configure", allNodes()),
		}),
		Entry("when no node matches policy node selector, policy state is not matching", ConditionsCase{
			Enactments:    []nmstatev1alpha1.NodeNetworkConfigurationEnactment{},
			NumberOfNodes: 3,
			Policy:        p(setPolicyNotMatching, "Policy does not match any node", forNode("node4")),
		}),
		Entry("when some enactments are from different profile it does no affect the profile status", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetSuccess),
				e("node1", "policy2", enactmentconditions.SetProgressing),
				e("node2", "policy2", enactmentconditions.SetProgressing),
			},
			NumberOfNodes: 3,
			Policy:        p(setPolicySuccess, "3/3 nodes successfully configured", allNodes()),
		}),
	)
})
