package handler

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/test/node"
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
			By(fmt.Sprintf("Deleteting linux bridge %s at all nodes", bridge1))
			setDesiredStateWithPolicyWithoutNodeSelector(bridge1, linuxBrAbsent(bridge1))
			waitForAvailablePolicy(bridge1)
			deletePolicy(bridge1)
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, resetPrimaryAndSecondaryNICs())
			waitForAvailableTestPolicy()
			deletePolicy(TestPolicy)

			By("Remove test label from node")
			node.RemoveLabels(nodes[0], testNodeSelector)
		})

		It("[test_id:3813]should not update any nodes and have false Matching state", func() {
			for _, node := range allNodes {
				enactmentConditionsStatusForPolicyEventually(node, bridge1).Should(ContainElement(
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
						Status: corev1.ConditionFalse,
					}))
			}
			for _, node := range allNodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
		})

		Context("and we remove the node selector", func() {
			BeforeEach(func() {
				By(fmt.Sprintf("Remove node selector at policy %s", bridge1))
				setDesiredStateWithPolicyWithoutNodeSelector(bridge1, linuxBrUp(bridge1))
				waitForAvailablePolicy(bridge1)
			})

			It("should update all nodes and have Matching enactment state", func() {
				for _, node := range allNodes {
					enactmentConditionsStatusForPolicyEventually(node, bridge1).Should(ContainElement(
						nmstate.Condition{
							Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
							Status: corev1.ConditionTrue,
						}))
				}
				for _, node := range allNodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				}

			})

		})
		Context("and we add the label to the node", func() {
			BeforeEach(func() {
				By("Add test label to node")
				node.AddLabels(nodes[0], testNodeSelector)
			})
			It("should apply the policy", func() {
				enactmentConditionsStatusForPolicyEventually(nodes[0], bridge1).Should(ContainElement(
					nmstate.Condition{
						Type:   nmstate.NodeNetworkConfigurationEnactmentConditionMatching,
						Status: corev1.ConditionTrue,
					}))
				//TODO: Remove this when webhook retest policy status when node labels are changed
				time.Sleep(3 * time.Second)
				waitForAvailablePolicy(bridge1)
				interfacesNameForNodeEventually(nodes[0]).Should(ContainElement(bridge1))
			})
		})
	})
})
