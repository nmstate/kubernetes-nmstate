package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/andreyvit/diff"

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

				obtainedStatus := shared.NodeNetworkStateStatus{}
				Consistently(func() shared.NodeNetworkStateStatus {
					obtainedStatus = nodeNetworkState(key).Status
					return obtainedStatus
				}, timeout, time.Second).Should(MatchAllFields(Fields{
					"CurrentState":             WithTransform(shared.State.String, Equal(originalNNS.Status.CurrentState.String())),
					"LastSuccessfulUpdateTime": Equal(originalNNS.Status.LastSuccessfulUpdateTime),
					"Conditions":               Equal(originalNNS.Status.Conditions),
				}), "currentState diff: ", diff.LineDiff(originalNNS.Status.CurrentState.String(), obtainedStatus.CurrentState.String()))
			}
		})
	})
	Context("when network configuration is changed by a NNCP", func() {
		BeforeEach(func() {
			// We want to test all the NNS so we apply policies to masters and workers
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
		It("should be immediately updated", func() {
			for node, originalNNS := range originalNNSs {
				key := types.NamespacedName{Name: node}

				Eventually(func() time.Time {
					updatedTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
					return updatedTime.Time
				}, time.Second*5, time.Second).Should(BeTemporally(">", originalNNS.Status.LastSuccessfulUpdateTime.Time))
			}
		})
	})

})
