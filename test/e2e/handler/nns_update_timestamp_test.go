package handler

import (
	"fmt"
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
		nnsByNode    = func() map[string]nmstatev1beta1.NodeNetworkState {
			nnss := map[string]nmstatev1beta1.NodeNetworkState{}
			for _, node := range allNodes {
				key := types.NamespacedName{Name: node}
				nnss[node] = nodeNetworkState(key)
			}
			return nnss
		}
	)
	Context("when network configuration hasn't change", func() {
		It("should not be updated", func() {
			Eventually(func() error {
				originalNNSs = nnsByNode()
				By("Give enough time for the NNS Reconcile to happen (3 interval times)")
				time.Sleep(2 * nmstatenode.NetworkStateRefresh)
				for node, originalNNS := range originalNNSs {
					obtainedNNSStatus := nodeNetworkState(types.NamespacedName{Name: node}).Status

					obtainedState := state.RemoveDynamicAttributesFromStruct(obtainedNNSStatus.CurrentState)
					originalState := state.RemoveDynamicAttributesFromStruct(originalNNS.Status.CurrentState)
					if obtainedState != originalState {
						return fmt.Errorf("should report same state, diff :%s", diff.LineDiff(originalState, obtainedState))
					}

					obtainedTimestamp := obtainedNNSStatus.LastSuccessfulUpdateTime
					originalTimestamp := originalNNS.Status.LastSuccessfulUpdateTime
					if obtainedTimestamp != originalTimestamp {
						return fmt.Errorf("should report same LastSuccessfulUpdateTime, diff :%s", diff.LineDiff(originalTimestamp.String(), obtainedTimestamp.String()))
					}
				}
				return nil
			}, nmstatenode.NetworkStateRefresh*4, nmstatenode.NetworkStateRefresh*2).ShouldNot(HaveOccurred())
		})
	})
	Context("when network configuration changed", func() {
		BeforeEach(func() {
			originalNNSs = nnsByNode()
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
				timeout := 2 * nmstatenode.NetworkStateRefresh
				key := types.NamespacedName{Name: node}

				Eventually(func() time.Time {
					updatedTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
					return updatedTime.Time
				}, timeout, time.Second).Should(BeTemporally(">", originalNNS.Status.LastSuccessfulUpdateTime.Time))
			}
		})
	})

})
