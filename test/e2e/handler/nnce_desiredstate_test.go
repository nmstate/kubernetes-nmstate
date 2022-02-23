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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("Enactment DesiredState", func() {
	Context("when applying a policy to matching nodes", func() {
		BeforeEach(func() {
			By("Create a policy")
			updateDesiredStateAndWait(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			By("Reset desired state at all nodes")
			resetDesiredStateForNodes()
		})
		It("should have desiredState for node", func() {
			for _, node := range nodes {
				enactmentKey := nmstate.EnactmentKey(node, TestPolicy)
				Byf("Check enactment %s has expected desired state", enactmentKey.Name)
				nnce := nodeNetworkConfigurationEnactment(enactmentKey)
				Expect(nnce.Status.DesiredState).To(MatchYAML(linuxBrUpWithDefaults(bridge1)))
			}
		})
	})
})
