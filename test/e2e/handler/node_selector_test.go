package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component]NodeSelector", func() {
	nonexistentNodeSelector := map[string]string{"nonexistentKey": "nonexistentValue"}

	Context("when policy is set with node selector not matching any nodes", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Set policy %s with not matching node selector", bridge1))
			setDesiredStateWithPolicyAndNodeSelector(bridge1, linuxBrUp(bridge1), nonexistentNodeSelector)
			waitForAvailablePolicy(bridge1)
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(bridge1, linuxBrAbsent(bridge1))
			waitForAvailablePolicy(bridge1)
			deletePolicy(bridge1)
			resetDesiredStateForNodes()
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
	})
})
