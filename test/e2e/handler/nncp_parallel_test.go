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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	corev1 "k8s.io/api/core/v1"
)

func enactmentsInProgress(policy string) int {
	progressingEnactments := 0
	for _, node := range nodes {
		enactment := enactmentConditionsStatus(node, policy)
		if cond := enactment.Find(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing); cond != nil {
			if cond.Status == corev1.ConditionTrue {
				progressingEnactments++
			}
		}
	}
	return progressingEnactments
}

var _ = Describe("NNCP with maxUnavailable", func() {
	duration := 15 * time.Second
	interval := 500 * time.Millisecond
	Context("when applying a policy to matching nodes", func() {
		BeforeEach(func() {
			By("Create a policy")
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Remove the policy")
			deletePolicy(TestPolicy)
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should be progressing on multiple nodes", func() {
			Eventually(func() int {
				return enactmentsInProgress(TestPolicy)
			}, duration, interval).Should(BeNumerically("==", maxUnavailableNodes()))
			waitForAvailablePolicy(TestPolicy)
		})
		It("should never exceed maxUnavailable nodes", func() {
			Consistently(func() int {
				return enactmentsInProgress(TestPolicy)
			}, duration, interval).Should(BeNumerically("<=", maxUnavailableNodes()))
			waitForAvailablePolicy(TestPolicy)
		})
	})
})
