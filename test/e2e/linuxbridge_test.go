package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
	createBridgeCr       = "deploy/crds/create-br1-linux-bridge.yaml"
	deleteBridgeCr       = "deploy/crds/delete-br1-linux-bridge.yaml"
	firstNodeName        = "node01" // TODO: Get it from cluster
	bridgeName           = "br1"
)

var _ = Describe("Linux Bridge", func() {
	Context("when created", func() {

		var (
			namespace      string
			ctx            *framework.TestCtx
			key            types.NamespacedName
			cleanupOptions *framework.CleanupOptions
		)

		BeforeEach(func() {
			ctx, namespace = prepare(t)
			key = types.NamespacedName{Namespace: namespace, Name: firstNodeName}
			cleanupOptions = &framework.CleanupOptions{
				TestContext:   ctx,
				Timeout:       cleanupTimeout,
				RetryInterval: cleanupRetryInterval}
		})

		It("should create the linux bridge successfully", func() {
			By("Creating the desiredState")
			createStateFromFile(createBridgeCr, namespace, cleanupOptions)

			By("Getting the currentState")
			currentState := currentState(key)
			Expect(currentState).ToNot(BeEmpty())
		})

		AfterEach(func() {

			// Apply the bridge deletion
			updateStateSpecFromFile(deleteBridgeCr, key)

			// Wait for bridge to be removed
			err := waitInterfaceDeleted(namespace, firstNodeName, bridgeName)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
