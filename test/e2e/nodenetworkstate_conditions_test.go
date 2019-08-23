package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeNetworkStateCondition", func() {
	var (
		br1Up = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)
		invalidConfig = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: afdshaf
`)
	)
	Context("when applying valid config", func() {
		BeforeEach(func() {
			updateDesiredState(br1Up)
		})
		AfterEach(func() {

			// First we clean desired state if we
			// don't do that nmstate recreates the bridge
			resetDesiredStateForNodes()

			// TODO: Add status conditions to ensure that
			//       it has being really reset so we can
			//       remove this ugly sleep
			time.Sleep(1 * time.Second)

			// Let's clean the bridge directly in the node
			// bypassing nmstate
			deleteConnectionAtNodes("eth1")
			deleteConnectionAtNodes("br1")
		})
		It("should have Available ConditionType set to true", func() {
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable, corev1.ConditionTrue)
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionFailing, corev1.ConditionFalse)
			}
		})
	})

	Context("when applying invalid configuration", func() {
		BeforeEach(func() {
			updateDesiredState(invalidConfig)
		})
		AfterEach(func() {

			// First we clean desired state if we
			// don't do that nmstate recreates the bridge
			resetDesiredStateForNodes()

			// TODO: Add status conditions to ensure that
			//       it has being really reset so we can
			//       remove this ugly sleep
			time.Sleep(1 * time.Second)

			// Let's clean the bridge directly in the node
			// bypassing nmstate
			deleteConnectionAtNodes("eth1")
		})
		It("should have Failing ConditionType set to true", func() {
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionFailing, corev1.ConditionTrue)
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable, corev1.ConditionFalse)
			}
		})
	})
})
