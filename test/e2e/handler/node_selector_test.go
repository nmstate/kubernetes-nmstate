package handler

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component]NodeSelector", func() {
	testNodeSelector := map[string]string{"testKey": "testValue"}
	Context("when policy is set with node selector not matching any nodes", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Set policy %s with not matching node selector", bridge1))
			setDesiredStateWithPolicyAndNodeSelector(bridge1, linuxBrUp(bridge1), testNodeSelector)
			waitForAvailablePolicy(bridge1)
		})

		AfterEach(func() {
			By(fmt.Sprintf("Deleteting linux bridge %s", bridge1))
			setDesiredStateWithPolicy(bridge1, linuxBrAbsent(bridge1))
			waitForAvailablePolicy(bridge1)
			deletePolicy(bridge1)
			resetDesiredStateForNodes()

			By("Remove test label from node")
			removeLabelsFromNode(nodes[0], testNodeSelector)
		})

		It("[test_id:3813]should not update any nodes and have false Matching state", func() {
			for _, node := range nodes {
				enactmentConditionsStatusForPolicyEventually(node, bridge1).Should(ContainElement(
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
						Status: corev1.ConditionFalse,
					}))
			}
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})

		Context("and we remove the node selector", func() {
			BeforeEach(func() {
				By(fmt.Sprintf("Remove node selector at policy %s", bridge1))
				setDesiredStateWithPolicyAndNodeSelector(bridge1, linuxBrUp(bridge1), map[string]string{})
				waitForAvailablePolicy(bridge1)
			})

			It("should update all nodes and have Matching enactment state", func() {
				for _, node := range nodes {
					enactmentConditionsStatusForPolicyEventually(node, bridge1).Should(ContainElement(
						nmstate.Condition{
							Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
							Status: corev1.ConditionTrue,
						}))
				}
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				}

			})

		})
		Context("and we add the label to the node", func() {
			BeforeEach(func() {
				By("Add test label to node")
				addLabelsToNode(nodes[0], testNodeSelector)
			})
			It("should apply the policy", func() {
				enactmentConditionsStatusForPolicyEventually(nodes[0], bridge1).Should(ContainElement(
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
						Status: corev1.ConditionTrue,
					}))
				waitForAvailablePolicy(bridge1)
				interfacesNameForNodeEventually(nodes[0]).Should(ContainElement(bridge1))
			})
		})
	})
})

func addLabelsToNode(nodeName string, labelsToAdd map[string]string) {
	node := corev1.Node{}
	err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving node to change labels")

	if len(node.Labels) == 0 {
		node.Labels = labelsToAdd
	} else {
		for k, v := range labelsToAdd {
			node.Labels[k] = v
		}
	}
	err = testenv.Client.Update(context.TODO(), &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success updating node with new labels")
}

func removeLabelsFromNode(nodeName string, labelsToRemove map[string]string) {
	node := corev1.Node{}
	err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving node to remove labels")

	if len(node.Labels) == 0 {
		return
	}

	for k, _ := range labelsToRemove {
		delete(node.Labels, k)
	}

	err = testenv.Client.Update(context.TODO(), &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success updating node with label delete")
}
