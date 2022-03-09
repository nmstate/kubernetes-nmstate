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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
)

const expectedDummyName = "dummy0"

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
	Context("when network configuration is not changed by a NNCP", func() {
		It("nns should not be updated after reconcile", func() {
			// Give enough time for the NNS to be reconcile(2 interval times)
			interval := 2 * nmstatenode.NetworkStateRefresh
			timeout := 4 * interval
			Eventually(func() error {
				for node, originalNNS := range originalNNSs {
					key := types.NamespacedName{Name: node}
					currentStatus := nodeNetworkState(key).Status
					originalStatus := originalNNS.Status
					if currentStatus.CurrentState.String() == originalStatus.CurrentState.String() {
						Byf("Check LastSuccessfulUpdateTime changed at %s", node)
						Expect(currentStatus.LastSuccessfulUpdateTime).To(Equal(originalStatus.LastSuccessfulUpdateTime))
					} else {
						return fmt.Errorf("Network configuration changed, sending and error to retry")
					}
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})
	Context("when network configuration is changed externally", func() {
		BeforeEach(func() {
			createDummyConnectionAtAllNodes(expectedDummyName)
		})
		AfterEach(func() {
			deleteConnectionAndWait(allNodes, expectedDummyName)
		})
		It("should update it with according to network state refresh duration", func() {
			for node, originalNNS := range originalNNSs {
				Byf("Checking timestamp against original one %s", originalNNS.Status.LastSuccessfulUpdateTime)
				Eventually(
					func() time.Time {
						currentNNS := nodeNetworkState(types.NamespacedName{Name: node})
						return currentNNS.Status.LastSuccessfulUpdateTime.Time
					},
					2*nmstatenode.NetworkStateRefresh,
					10*time.Second,
				).Should(
					BeTemporally(">", originalNNS.Status.LastSuccessfulUpdateTime.Time), "should update it at %s", node,
				)
			}
		})

	})
	Context("when network configuration is changed by a NNCP", func() {
		BeforeEach(func() {
			// We want to test all the NNS so we apply policies to control-plane and workers
			// (use linuxBrUpNoPorts to not affect the nodes secondary interfaces state)
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, linuxBrUpNoPorts(bridge1))
			waitForAvailableTestPolicy()
		})
		AfterEach(func() {
			setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, linuxBrAbsent(bridge1))
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
