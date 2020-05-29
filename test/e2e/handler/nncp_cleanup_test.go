package handler

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/shared"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NNCP cleanup", func() {

	Context("when a policy is deleted", func() {
		BeforeEach(func() {
			By("Create a policy")
			setDesiredStateWithPolicy(bridge1, linuxBrUp(bridge1))

			By("Wait for policy to be ready")
			waitForAvailablePolicy(bridge1)

			By("Delete the policy")
			deletePolicy(bridge1)
		})

		AfterEach(func() {
			deletePolicy(bridge1)
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})

		It("should also delete nodes enactments", func() {
			for _, node := range nodes {
				Eventually(func() bool {
					key := nmstate.EnactmentKey(node, bridge1)
					enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
					err := framework.Global.Client.Get(context.TODO(), key, &enactment)
					return errors.IsNotFound(err)
				}, 10*time.Second, 1*time.Second).Should(BeTrue(), "Enactment has not being deleted")
			}
		})
	})
})
