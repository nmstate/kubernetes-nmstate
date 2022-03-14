/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactment"
)

var _ = Describe("NodeSelector", func() {
	var (
		testNodeSelector            = map[string]string{"testKey": "testValue"}
		numberOfEnactmentsForPolicy = func(policyName string) int {
			nncp := nodeNetworkConfigurationPolicy(policyName)
			numberOfMatchingEnactments, _, err := enactment.CountByPolicy(testenv.Client, &nncp)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return numberOfMatchingEnactments
		}
	)
	Context("when policy is set with node selector not matching any nodes", func() {
		BeforeEach(func() {
			Byf("Set policy %s with not matching node selector", bridge1)
			// use linuxBrUpNoPorts to not affect the nodes secondary interfaces state
			setDesiredStateWithPolicyAndNodeSelectorEventually(bridge1, linuxBrUpNoPorts(bridge1), testNodeSelector)
			waitForAvailablePolicy(bridge1)
		})

		AfterEach(func() {
			Byf("Deleteting linux bridge %s at all nodes", bridge1)
			setDesiredStateWithPolicyWithoutNodeSelector(bridge1, linuxBrAbsent(bridge1))
			waitForAvailablePolicy(bridge1)
			deletePolicy(bridge1)
		})

		It("should not update any nodes and have not enactments", func() {
			for _, node := range allNodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			Expect(numberOfEnactmentsForPolicy(bridge1)).To(Equal(0), "should not create any enactment")
		})

		Context("and we remove the node selector", func() {
			BeforeEach(func() {
				Byf("Remove node selector at policy %s", bridge1)
				// use linuxBrUpNoPorts to not affect the nodes secondary interfaces state
				setDesiredStateWithPolicyWithoutNodeSelector(bridge1, linuxBrUpNoPorts(bridge1))
				waitForAvailablePolicy(bridge1)
			})

			It("should update all nodes and have Matching enactment state", func() {
				for _, node := range allNodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				}
				Expect(numberOfEnactmentsForPolicy(bridge1)).To(Equal(len(allNodes)), "should create all the enactments")

			})

		})
		Context("and we add the label to the node", func() {
			BeforeEach(func() {
				By("Add test label to node")
				addLabelsToNode(nodes[0], testNodeSelector)
				//TODO: Remove this when webhook retest policy status when node labels are changed
				time.Sleep(3 * time.Second)
				waitForAvailablePolicy(bridge1)
			})
			AfterEach(func() {
				By("Remove test label from node")
				removeLabelsFromNode(nodes[0], testNodeSelector)
			})
			It("should apply the policy", func() {
				By("Check that NNCE is created")
				nodeNetworkConfigurationEnactment(shared.EnactmentKey(nodes[0], bridge1))
				interfacesNameForNodeEventually(nodes[0]).Should(ContainElement(bridge1))
			})
			Context("and remove the label again", func() {
				BeforeEach(func() {
					removeLabelsFromNode(nodes[0], testNodeSelector)
					//TODO: Remove this when webhook retest policy status when node labels are changed
					time.Sleep(3 * time.Second)
					waitForAvailablePolicy(bridge1)
				})
				It("should remove the not matching enactment", func() {
					Expect(numberOfEnactmentsForPolicy(bridge1)).To(Equal(0), "should remove the not matching enactment")
				})
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

	for k := range labelsToRemove {
		delete(node.Labels, k)
	}

	err = testenv.Client.Update(context.TODO(), &node)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success updating node with label delete")
}
