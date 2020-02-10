package e2e

import (
	"testing"
	"time"

	"github.com/pkg/errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

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
	By("Check operator is up and running")
	namespace, err := ctx.GetNamespace()
	Expect(err).ToNot(HaveOccurred())
	err = waitForDaemonSets(t, framework.Global.KubeClient, namespace, time.Second*5, time.Second*90)
	Expect(err).ToNot(HaveOccurred(), "operator is not ready")
	return ctx, namespace
}

func waitForDaemonSets(t *testing.T, kubeclient kubernetes.Interface, namespace string, retryInterval, timeout time.Duration) error {
	if framework.Global.LocalOperator {
		return nil
	}
	err := wait.PollImmediate(retryInterval, timeout, func() (done bool, err error) {
		daemonsets, err := kubeclient.AppsV1().DaemonSets(namespace).List(metav1.ListOptions{})
		if err != nil {
			return true, errors.Wrapf(err, "failed retrieving daemon sets for namespace %s", namespace)
		}
		for _, daemonset := range daemonsets.Items {
			if daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberAvailable {
				return false, nil
			}
		}
		return true, nil
	})
	return err
}
