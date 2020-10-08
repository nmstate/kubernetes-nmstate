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

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/enactmentstatus/conditions"
)

func e(node string, policy string, conditionsSetters ...func(*nmstate.ConditionList, string)) nmstatev1beta1.NodeNetworkConfigurationEnactment {
	conditions := nmstate.ConditionList{}
	for _, conditionsSetter := range conditionsSetters {
		conditionsSetter(&conditions, "")
	}
	return nmstatev1beta1.NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				nmstate.EnactmentPolicyLabel: policy,
			},
			Name: nmstate.EnactmentKey(node, policy).Name,
		},
		Status: nmstate.NodeNetworkConfigurationEnactmentStatus{
			Conditions: conditions,
		},
	}
}

func p(conditionsSetter func(*nmstate.ConditionList, string), message string) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	conditions := nmstate.ConditionList{}
	conditionsSetter(&conditions, message)
	return nmstatev1beta1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "policy1",
		},
		Status: nmstate.NodeNetworkConfigurationPolicyStatus{
			Conditions: conditions,
		},
	}
}

func nodeName(idx int) string {
	return fmt.Sprintf("node%d", idx)
}

func newNode(idx int) corev1.Node {
	nodeName := nodeName(idx)
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"kubernetes.io/hostname": nodeName,
			},
		},
	}
	return node
}

func newPodAtNode(idx int, name string, namespace string, app string) corev1.Pod {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", name, idx),
			Namespace: namespace,
			Labels: map[string]string{
				"app": app,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName(idx),
		},
	}
	return pod
}

func newNmstatePodAtNode(idx int) corev1.Pod {
	return newPodAtNode(idx, "nmstate-handler", "nmstate", "kubernetes-nmstate")
}

func newNonNmstatePodAtNode(idx int) corev1.Pod {
	return newPodAtNode(idx, "foo-name", "foo-namespace", "foo-app")
}

func newNmstatePods(cardinality int) []corev1.Pod {
	pods := []corev1.Pod{}
	for i := 1; i <= cardinality; i++ {
		pods = append(pods, newNmstatePodAtNode(i))
	}
	return pods
}

func newNodes(cardinality int) []corev1.Node {
	nodes := []corev1.Node{}
	for i := 1; i <= cardinality; i++ {
		nodes = append(nodes, newNode(i))
	}
	return nodes
}

func cleanTimestamps(conditions nmstate.ConditionList) nmstate.ConditionList {
	dummyTime := metav1.Time{Time: time.Unix(0, 0)}
	for i, _ := range conditions {
		conditions[i].LastHeartbeatTime = dummyTime
		conditions[i].LastTransitionTime = dummyTime
	}
	return conditions
}

var _ = Describe("Policy Conditions", func() {
	type ConditionsCase struct {
		Enactments []nmstatev1beta1.NodeNetworkConfigurationEnactment
		Nodes      []corev1.Node
		Policy     nmstatev1beta1.NodeNetworkConfigurationPolicy
		Pods       []corev1.Pod
	}
	DescribeTable("the policy overall condition",
		func(c ConditionsCase) {
			objs := []runtime.Object{}
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.SchemeGroupVersion,
				&nmstatev1beta1.NodeNetworkConfigurationPolicy{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{},
			)

			for i, _ := range c.Enactments {
				// We cannot use the memory from the element
				// returned by range, since it's has the same
				// memory address it will be added multiple time
				// with duplicated values
				objs = append(objs, &c.Enactments[i])
			}
			for i, _ := range c.Nodes {
				objs = append(objs, &c.Nodes[i])
			}
			for i, _ := range c.Pods {
				objs = append(objs, &c.Pods[i])
			}

			updatedPolicy := c.Policy.DeepCopy()
			updatedPolicy.Status.Conditions = nmstate.ConditionList{}

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
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 0/3 nodes finished"),
		}),
		Entry("when all enactments are success then policy is success", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
		Entry("when not all enactments are created is progressing", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(4),
			Pods:   newNmstatePods(4),
			Policy: p(setPolicyProgressing, "Policy is progressing 3/4 nodes finished"),
		}),
		Entry("when enactments are progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 2/3 nodes finished"),
		}),
		Entry("when enactments are failed/progressing/success then policy is progressing", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node4", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(4),
			Pods:   newNmstatePods(4),
			Policy: p(setPolicyProgressing, "Policy is progressing 3/4 nodes finished"),
		}),
		Entry("when all the enactments are at failing or success policy is degraded", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyFailedToConfigure, "2/3 nodes failed to configure"),
		}),
		Entry("when all the enactments are at failing policy is degraded", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetFailedToConfigure),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyFailedToConfigure, "3/3 nodes failed to configure"),
		}),
		Entry("when no node matches policy node selector, policy state is not matching", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
				e("node2", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
				e("node3", "policy1", enactmentconditions.SetNodeSelectorNotMatching),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyNotMatching, "Policy does not match any node"),
		}),
		Entry("when some enacments has unknown matching state policy state is progressing", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1"),
				e("node2", "policy1"),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicyProgressing, "Policy is progressing 1/3 nodes finished"),
		}),
		Entry("when some enactments are from different profile it does no affect the profile status", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node1", "policy2", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
				e("node2", "policy2", enactmentconditions.SetMatching, enactmentconditions.SetProgressing),
			},
			Nodes:  newNodes(3),
			Pods:   newNmstatePods(3),
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
		Entry("when a node does not run nmstate pod ignore it for policy conditions calculations", ConditionsCase{
			Enactments: []nmstatev1beta1.NodeNetworkConfigurationEnactment{
				e("node1", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node2", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
				e("node3", "policy1", enactmentconditions.SetMatching, enactmentconditions.SetSuccess),
			},
			Nodes: newNodes(4),
			Pods: []corev1.Pod{
				newNmstatePodAtNode(1),
				newNonNmstatePodAtNode(1),
				newNmstatePodAtNode(2),
				newNonNmstatePodAtNode(2),
				newNmstatePodAtNode(3),
				newNonNmstatePodAtNode(3),
				newNonNmstatePodAtNode(4),
			},
			Policy: p(setPolicySuccess, "3/3 nodes successfully configured"),
		}),
	)
})
