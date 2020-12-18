package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
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
			for node, originalNNS := range originalNNSs {
				// Give enough time for the NNS to be updated (3 interval times)
				timeout := 3 * nmstatenode.NetworkStateRefresh
				key := types.NamespacedName{Name: node}

				Consistently(func() shared.NodeNetworkStateStatus {
					return nodeNetworkState(key).Status
				}, timeout, time.Second).Should(Equal(originalNNS.Status))
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
