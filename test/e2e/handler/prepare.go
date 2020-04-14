package handler

import (
	"fmt"
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
	Expect(err).ToNot(HaveOccurred(), "operator daemonset is not ready")
	err = waitForDeployments(t, framework.Global.KubeClient, namespace, time.Second*5, time.Second*90)
	Expect(err).ToNot(HaveOccurred(), "operator deployment is not ready")
	return ctx, namespace
}

func waitForDaemonSets(t *testing.T, kubeclient kubernetes.Interface, namespace string, retryInterval, timeout time.Duration) error {
	if framework.Global.LocalOperator {
		return nil
	}
	err := wait.PollImmediate(retryInterval, timeout, func() (done bool, err error) {
		filterByApp := metav1.ListOptions{LabelSelector: "app=kubernetes-nmstate"}
		daemonsets, err := kubeclient.AppsV1().DaemonSets(namespace).List(filterByApp)
		if err != nil {
			return true, errors.Wrapf(err, "failed retrieving daemon sets for namespace %s", namespace)
		}
		for _, daemonset := range daemonsets.Items {
			By(fmt.Sprintf("Checking daemonset %s", daemonset.Name))
			if daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberAvailable {
				return false, nil
			}
		}
		return true, nil
	})
	return err
}

func waitForDeployments(t *testing.T, kubeclient kubernetes.Interface, namespace string, retryInterval, timeout time.Duration) error {
	if framework.Global.LocalOperator {
		return nil
	}
	err := wait.PollImmediate(retryInterval, timeout, func() (done bool, err error) {
		filterByApp := metav1.ListOptions{LabelSelector: "app=kubernetes-nmstate"}
		deployments, err := kubeclient.AppsV1().Deployments(namespace).List(filterByApp)
		if err != nil {
			return true, errors.Wrapf(err, "failed retrieving daemon sets for namespace %s", namespace)
		}
		for _, deployment := range deployments.Items {
			By(fmt.Sprintf("Checking deployment %s", deployment.Name))
			if int(deployment.Status.AvailableReplicas) >= 2 {
				return true, nil
			}
		}
		return true, nil
	})
	return err
}
