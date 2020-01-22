package policyconditions

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/controller/nodenetworkconfigurationpolicy/enactmentstatus/conditions"
)

func e(node string, policy string, conditionsSetters ...func(*nmstatev1alpha1.ConditionList, string)) nmstatev1alpha1.NodeNetworkConfigurationEnactment {
	conditions := nmstatev1alpha1.ConditionList{}
	for _, conditionsSetter := range conditionsSetters {
		conditionsSetter(&conditions, "")
	}
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

func p(conditionsSetter func(*nmstatev1alpha1.ConditionList, string), message string) nmstatev1alpha1.NodeNetworkConfigurationPolicy {
	conditions := nmstatev1alpha1.ConditionList{}
	conditionsSetter(&conditions, message)
	return nmstatev1alpha1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "policy1",
		},
		Status: nmstatev1alpha1.NodeNetworkConfigurationPolicyStatus{
			Conditions: conditions,
		},
	}
}

func newNode(idx int, conditions []corev1.NodeCondition) corev1.Node {
	nodeName := fmt.Sprintf("node%d", idx)
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname": nodeName,
			},
		},
		Status: corev1.NodeStatus{
			Conditions: conditions,
		},
	}
	return node
}

func nodeReady() []corev1.NodeCondition {
	return []corev1.NodeCondition{
		corev1.NodeCondition{
			Type:   corev1.NodeReady,
			Status: corev1.ConditionTrue,
		},
	}
}

func nodeNotReady() []corev1.NodeCondition {
	return []corev1.NodeCondition{
		corev1.NodeCondition{
			Type:   corev1.NodeReady,
			Status: corev1.ConditionFalse,
		},
	}
}

func newReadyNodes(cardinality int) []corev1.Node {
	nodes := []corev1.Node{}
	for i := 1; i <= cardinality; i++ {
		nodes = append(nodes, newNode(i, nodeReady()))
	}
	return nodes
}

func cleanTimestamps(conditions nmstatev1alpha1.ConditionList) nmstatev1alpha1.ConditionList {
	dummyTime := metav1.Time{Time: time.Unix(0, 0)}
	for i, _ := range conditions {
		conditions[i].LastHeartbeatTime = dummyTime
		conditions[i].LastTransitionTime = dummyTime
	}
	return conditions
}

var _ = Describe("Policy Conditions", func() {
	type ConditionsCase struct {
		Enactments []nmstatev1alpha1.NodeNetworkConfigurationEnactment
		Nodes      []corev1.Node
		Policy     nmstatev1alpha1.NodeNetworkConfigurationPolicy
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
			for i, _ := range c.Nodes {
				objs = append(objs, &c.Nodes[i])
			}

			updatedPolicy := c.Policy.DeepCopy()
			updatedPolicy.Status.Conditions = nmstatev1alpha1.ConditionList{}

			objs = append(objs, updatedPolicy)

			client := fake.NewFakeClientWithScheme(s, objs...)
			key := types.NamespacedName{Name: updatedPolicy.Name}
			err := Update(client, key)
			Expect(err).ToNot(HaveOccurred())
			err = client.Get(context.TODO(), key, updatedPolicy)
			Expect(err).ToNot(HaveOccurred())
			Expect(cleanTimestamps(updatedPolicy.Status.Conditions)).To(ConsistOf(cleanTimestamps(c.Policy.Status.Conditions)))
		},
		Entry("when all enactments are progressing then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 0/3 nodes finished"),
		}),
		Entry("when all enactments are success then policy is success", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
		Entry("when not all enactments are created is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(4),
			Policy: p(setPolicyProgressing, "Policy is progressing 3/4 nodes finished"),
		}),
		Entry("when enactments are progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 2/3 nodes finished"),
		}),
		Entry("when enactments are failed/progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node4", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(4),
			Policy: p(setPolicyProgressing, "Policy is progressing 3/4 nodes finished"),
		}),
		Entry("when all the enactments are at failing or success policy is degraded", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyFailedToConfigure, "2/3 nodes failed to configure"),
		}),
		Entry("when all the enactments are at failing policy is degraded", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyFailedToConfigure, "3/3 nodes failed to configure"),
		}),
		Entry("when no node matches policy node selector, policy state is not matching", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
				e("node2", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
				e("node3", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyNotMatching, "Policy does not match any node"),
		}),
		Entry("when some enacments has unknown matching state policy state is progressing", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1"),
				e("node2", "policy1"),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 1/3 nodes finished"),
		}),
		Entry("when some enactments are from different profile it does no affect the profile status", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node1", "policy2", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node2", "policy2", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
			},
			Nodes:  newReadyNodes(3),
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
		Entry("when a node is not ready ignore it for policy conditions calculations", ConditionsCase{
			Enactments: []nmstatev1alpha1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes: []corev1.Node{
				newNode(1, nodeReady()),
				newNode(2, nodeReady()),
				newNode(3, nodeReady()),
				newNode(4, nodeNotReady()),
			},
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
	)
})
