package e2e

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	yaml "sigs.k8s.io/yaml"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

func writePodsLogs(namespace string, writer io.Writer) error {
	if framework.Global.LocalOperator {
		return nil
	}
	podLogOpts := corev1.PodLogOptions{}
	podList := &corev1.PodList{}
	err := framework.Global.Client.List(context.TODO(), &dynclient.ListOptions{}, podList)
	Expect(err).ToNot(HaveOccurred())
	podsClientset := framework.Global.KubeClient.CoreV1().Pods(namespace)

	for _, pod := range podList.Items {
		appLabel, hasAppLabel := pod.Labels["app"]
		if !hasAppLabel || appLabel != "kubernetes-nmstate" {
			continue
		}
		req := podsClientset.GetLogs(pod.Name, &podLogOpts)
		podLogs, err := req.Stream()
		if err != nil {
			io.WriteString(writer, fmt.Sprintf("error in opening stream: %v\n", err))
			continue
		}
		defer podLogs.Close()
		_, err = io.Copy(writer, podLogs)
		if err != nil {
			io.WriteString(writer, fmt.Sprintf("error in copy information from podLogs to buf: %v\n", err))
			continue
		}

	}
	return nil
}

func interfacesName(interfaces []interface{}) []string {
	var names []string
	for _, iface := range interfaces {
		name, hasName := iface.(map[string]interface{})["name"]
		Expect(hasName).To(BeTrue(), "should have name field in the interfaces, https://github.com/nmstate/nmstate/blob/master/libnmstate/schemas/operational-state.yaml")
		names = append(names, name.(string))
	}
	return names
}

func prepare(t *testing.T) (*framework.TestCtx, string) {
	By("Initialize cluster resources")
	cleanupRetryInterval := time.Second * 1
	cleanupTimeout := time.Second * 5
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	Expect(err).ToNot(HaveOccurred())

	// get namespace
	namespace, err := ctx.GetNamespace()
	Expect(err).ToNot(HaveOccurred())

	err = WaitForOperatorDaemonSet(t, framework.Global.KubeClient, namespace, "nmstate-handler", time.Second*5, time.Second*90)
	Expect(err).ToNot(HaveOccurred())
	return ctx, namespace
}

// WaitForOperatorDeployment has the same functionality as WaitForDeployment but will no wait for the deployment if the
// test was run with a locally run operator (--up-local flag)
func WaitForOperatorDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	return waitForDaemonSet(t, kubeclient, namespace, name, retryInterval, timeout, true)
}

func waitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration, isOperator bool) error {
	if isOperator && framework.Global.LocalOperator {
		t.Log("Operator is running locally; skip waitForDeployment")
		return nil
	}
	err := wait.PollImmediate(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}

		if deployment.Status.DesiredNumberScheduled == deployment.Status.NumberAvailable {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s daemonset (%d/%d)\n", name, deployment.Status.DesiredNumberScheduled, deployment.Status.NumberAvailable)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Log("DaemonSet available")
	return nil
}

func readStateFromFile(file string, namespace string) nmstatev1.NodeNetworkState {
	manifest, err := ioutil.ReadFile(file)
	Expect(err).ToNot(HaveOccurred())

	state := nmstatev1.NodeNetworkState{}
	err = yaml.Unmarshal(manifest, &state)
	Expect(err).ToNot(HaveOccurred())
	state.ObjectMeta.Namespace = namespace
	return state
}

func createStateFromFile(file string, namespace string, cleanupOptions *framework.CleanupOptions) {
	state := readStateFromFile(file, namespace)
	err := framework.Global.Client.Create(context.TODO(), &state, cleanupOptions)
	Expect(err).ToNot(HaveOccurred())
}

func updateStateSpecFromFile(file string, key types.NamespacedName) {
	state := nmstatev1.NodeNetworkState{}
	stateFromManifest := readStateFromFile(file, key.Namespace)
	err := framework.Global.Client.Get(context.TODO(), key, &state)
	Expect(err).ToNot(HaveOccurred())
	state.Spec = stateFromManifest.Spec
	err = framework.Global.Client.Update(context.TODO(), &state)
	Expect(err).ToNot(HaveOccurred())
}

func nodeNetworkState(key types.NamespacedName) nmstatev1.NodeNetworkState {
	state := nmstatev1.NodeNetworkState{}
	timeout := 5 * time.Second
	interval := 1 * time.Second
	Eventually(func() error {
		return framework.Global.Client.Get(context.TODO(), key, &state)
	}, timeout, interval).ShouldNot(HaveOccurred())
	return state
}

func deleteNodeNeworkStates() {
	nodeNetworkStateList := &nmstatev1.NodeNetworkStateList{}
	err := framework.Global.Client.List(context.TODO(), &dynclient.ListOptions{}, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())
	var deleteErrors []error
	for _, nodeNetworkState := range nodeNetworkStateList.Items {
		deleteErrors = append(deleteErrors, framework.Global.Client.Delete(context.TODO(), &nodeNetworkState))
	}
	Expect(deleteErrors).ToNot(ContainElement(HaveOccurred()))
}
