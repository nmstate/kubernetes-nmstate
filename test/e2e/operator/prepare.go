package operator

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

func prepare(t *testing.T) (*framework.TestCtx, string) {
	By("Initialize cluster resources")
	cleanupRetryInterval := time.Second * 1
	cleanupTimeout := time.Second * 5
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	Expect(err).ToNot(HaveOccurred(), "cluster resources not initialized")

	// get namespace
	By("Check operator is up and running")
	namespace, err := ctx.GetNamespace()
	Expect(err).ToNot(HaveOccurred())
	err = e2eutil.WaitForDeployment(t, framework.Global.KubeClient, namespace, "nmstate-operator", 1, time.Second*5, time.Second*90)
	Expect(err).ToNot(HaveOccurred(), "operator is not ready")
	return ctx, namespace
}
