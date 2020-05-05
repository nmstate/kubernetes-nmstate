package operator

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

func prepare(t *testing.T) (*framework.TestCtx, string) {
	By("Initialize cluster resources")
	cleanupRetryInterval := time.Second * 1
	cleanupTimeout := time.Second * 5
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	Expect(err).ToNot(HaveOccurred(), "cluster resources not initialized")

	// get namespace
	By("Get namespace")
	namespace, err := ctx.GetNamespace()
	Expect(err).ToNot(HaveOccurred())
	return ctx, namespace
}
