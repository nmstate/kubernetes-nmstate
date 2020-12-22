package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"github.com/andreyvit/diff"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
)

var _ = Describe("[nns] NNS LastSuccessfulUpdateTime", func() {
	var (
		originalNNSs map[string]nmstatev1beta1.NodeNetworkState
	)
	BeforeEach(func() {
		originalNNSs = map[string]nmstatev1beta1.NodeNetworkState{}
		for _, node := range allNodes {
			key := types.NamespacedName{Name: node}
			originalNNSs[node] = nodeNetworkState(key)
		}
	})
	Context("when network configuration hasn't change", func() {
		It("should not be updated", func() {
			By("Give enough time for the NNS Reconcile to happend (3 interval times)")
			time.Sleep(3 * nmstatenode.NetworkStateRefresh)
			for node, originalNNS := range originalNNSs {
				obtainedNNSStatus := nodeNetworkState(types.NamespacedName{Name: node}).Status
				obtainedState := state.RemoveDynamicAttributesFromStruct(obtainedNNSStatus.CurrentState)
				originalState := state.RemoveDynamicAttributesFromStruct(originalNNS.Status.CurrentState)
				Expect(obtainedState).To(Equal(originalState), "should report same state, diff :%s", diff.LineDiff(originalState, obtainedState))
				Expect(obtainedNNSStatus.LastSuccessfulUpdateTime).To(Equal(originalNNS.Status.LastSuccessfulUpdateTime))
				Expect(obtainedNNSStatus.Conditions).To(Equal(originalNNS.Status.Conditions))
			}
		})
	})
	Context("when network configuration changed", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, linuxBrUp(bridge1))
			waitForAvailableTestPolicy()
		})
		AfterEach(func() {
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, linuxBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, resetPrimaryAndSecondaryNICs())
			waitForAvailableTestPolicy()
			deletePolicy(TestPolicy)
		})
		It("should be updated", func() {
			for node, originalNNS := range originalNNSs {
				// Give enough time for the NNS to be updated (3 interval times)
				timeout := 3 * nmstatenode.NetworkStateRefresh
				key := types.NamespacedName{Name: node}

				Eventually(func() time.Time {
					updatedTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
					return updatedTime.Time
				}, timeout, time.Second).Should(BeTemporally(">", originalNNS.Status.LastSuccessfulUpdateTime.Time))
			}
		})
	})

})
