package handler

import (
	"fmt"

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
				By(fmt.Sprintf("Check enactment %s has expected desired state", enactmentKey.Name))
				nnce := nodeNetworkConfigurationEnactment(enactmentKey)
				Expect(nnce.Status.DesiredState).To(MatchYAML(linuxBrUp(bridge1)))
			}
		})
	})
})
