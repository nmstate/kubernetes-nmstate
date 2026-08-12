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

	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	policyconditions "github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

// deleteHandlerPodOnNode deletes the nmstate-handler pod running on the given
// node, simulating a handler death mid-apply. The DaemonSet recreates it.
func deleteHandlerPodOnNode(node string) {
	Byf("Deleting nmstate-handler pod on node %s", node)
	podList := corev1.PodList{}
	filterHandlers := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
	err := testenv.Client.List(context.TODO(), &podList, filterHandlers, client.InNamespace(testenv.OperatorNamespace))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	deleted := false
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Spec.NodeName == node {
			ExpectWithOffset(1, testenv.Client.Delete(context.TODO(), pod)).To(Succeed())
			deleted = true
		}
	}
	ExpectWithOffset(1, deleted).To(BeTrue(), "no nmstate-handler pod found on node %s", node)
}

// Regression test for OCPBUGS-74261: a handler killed mid-apply must not
// leave the policy permanently blocked on MaxUnavailableLimitReached. The
// replacement handler pod reclaims the ghost unavailable slot at startup and
// the audit recomputes the unavailable node count, so the policy converges.
var _ = Describe("NNCP unavailable-slot recovery after handler death", func() {
	Context("when the nmstate-handler pod is deleted while a policy is applying", func() {
		BeforeEach(func() {
			By("Create a policy that touches all test nodes")
			updateDesiredState(linuxBrUp(bridge1))

			By("Waiting for a node whose enactment is Progressing")
			var progressingNode string
			Eventually(func() string {
				for _, node := range nodes {
					enactment := policyconditions.EnactmentConditionsStatus(node, TestPolicy)
					condProgressing := enactment.Find(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing)
					if condProgressing != nil && condProgressing.Status == corev1.ConditionTrue {
						progressingNode = node
						return progressingNode
					}
				}
				return ""
			}, 15*time.Second, 500*time.Millisecond).ShouldNot(BeEmpty(),
				"no node reached Progressing for policy %s", TestPolicy)

			By("Killing the handler pod on the progressing node while the policy progresses")
			deleteHandlerPodOnNode(progressingNode)
		})
		AfterEach(func() {
			// Policy deletion and node reset must run even when the
			// absent-wait fails (e.g. the policy is stuck), otherwise the
			// policy leaks into subsequent specs.
			defer func() {
				By("Remove the policy")
				deletePolicy(TestPolicy)
				By("Reset desired state at all nodes")
				resetDesiredStateForNodes()
			}()
			By("Remove the bridge")
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
		})
		It("should eventually reach Available without recreating the policy", func() {
			policyconditions.WaitForAvailablePolicy(TestPolicy)

			By("Verifying no ghost unavailable slots remain")
			nncp := nodeNetworkConfigurationPolicy(TestPolicy)
			for generation, count := range nncp.Status.UnavailableNodeCountMap {
				Expect(count).To(Equal(0), "generation %s should have no unavailable slots", generation)
			}
		})
	})
})
