package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
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

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const ReadTimeout = 15 * time.Second
const ReadInterval = 1 * time.Second

func writePodsLogs(namespace string, sinceTime time.Time, writer io.Writer) error {
	if framework.Global.LocalOperator {
		return nil
	}

	podLogOpts := corev1.PodLogOptions{}
	podLogOpts.SinceTime = &metav1.Time{sinceTime}
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

func interfaceByName(interfaces []interface{}, searchedName string) map[string]interface{} {
	var dummy map[string]interface{}
	for _, iface := range interfaces {
		name, hasName := iface.(map[string]interface{})["name"]
		Expect(hasName).To(BeTrue(), "should have name field in the interfaces, https://github.com/nmstate/nmstate/blob/master/libnmstate/schemas/operational-state.yaml")
		if name == searchedName {
			return iface.(map[string]interface{})
		}
	}
	Fail(fmt.Sprintf("interface %s not found at %+v", searchedName, interfaces))
	return dummy
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

func updateDesiredStateAtNode(node string, desiredState nmstatev1alpha1.State) {
	key := types.NamespacedName{Name: node}
	state := nmstatev1alpha1.NodeNetworkState{}
	err := framework.Global.Client.Get(context.TODO(), key, &state)
	Expect(err).ToNot(HaveOccurred())
	state.Spec.DesiredState = desiredState
	err = framework.Global.Client.Update(context.TODO(), &state)
	Expect(err).ToNot(HaveOccurred())
}

func updateDesiredState(desiredState nmstatev1alpha1.State) {
	for _, node := range nodes {
		updateDesiredStateAtNode(node, desiredState)
	}
}

func resetDesiredStateForNodes() {
	for _, node := range nodes {
		updateDesiredStateAtNode(node, nmstatev1alpha1.State(""))
	}
}

func nodeNetworkState(key types.NamespacedName) nmstatev1alpha1.NodeNetworkState {
	state := nmstatev1alpha1.NodeNetworkState{}
	Eventually(func() error {
		return framework.Global.Client.Get(context.TODO(), key, &state)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return state
}

func deleteNodeNeworkStates() {
	nodeNetworkStateList := &nmstatev1alpha1.NodeNetworkStateList{}
	err := framework.Global.Client.List(context.TODO(), &dynclient.ListOptions{}, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())
	var deleteErrors []error
	for _, nodeNetworkState := range nodeNetworkStateList.Items {
		deleteErrors = append(deleteErrors, framework.Global.Client.Delete(context.TODO(), &nodeNetworkState))
	}
	Expect(deleteErrors).ToNot(ContainElement(HaveOccurred()))
}

func run(node string, command ...string) error {
	ssh_command := []string{node}
	ssh_command = append(ssh_command, command...)
	cmd := exec.Command("./kubevirtci/cluster-up/ssh.sh", ssh_command...)
	GinkgoWriter.Write([]byte(strings.Join(ssh_command, " ") + "\n"))
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	GinkgoWriter.Write([]byte(stdout.String() + stderr.String() + "\n"))
	return err
}

func runAtNodes(command ...string) (errs []error) {
	for _, node := range nodes {
		errs = append(errs, run(node, command...))
	}
	return errs
}

func deleteBridgeAtNodes(bridgeName string) []error {
	By(fmt.Sprintf("Delete bridge %s", bridgeName))
	return runAtNodes("sudo", "nmcli", "con", "delete", bridgeName)
}

func createBridgeAtNodes(bridgeName string) []error {
	By(fmt.Sprintf("Creating bridge %s", bridgeName))
	return runAtNodes("sudo", "nmcli", "con", "add", "type", "bridge", "ifname", bridgeName)
}

func createDummyAtNodes(dummyName string) []error {
	By(fmt.Sprintf("Creating dummy %s", dummyName))
	return runAtNodes("sudo", "nmcli", "con", "add", "type", "dummy", "con-name", dummyName, "ifname", dummyName)
}

func deleteConnectionAtNodes(name string) []error {
	By(fmt.Sprintf("Delete connection %s", name))
	return runAtNodes("sudo", "nmcli", "con", "delete", name)
}

func interfaces(state nmstatev1alpha1.State) []interface{} {
	By("unmarshal state yaml into unstructured golang")
	var stateUnstructured map[string]interface{}
	err := yaml.Unmarshal(state, &stateUnstructured)
	Expect(err).ToNot(HaveOccurred(), "Should parse correctly yaml: %s", state)
	interfaces := stateUnstructured["interfaces"].([]interface{})
	return interfaces
}

func currentState(namespace string, node string, currentStateYaml *nmstatev1alpha1.State) AsyncAssertion {
	key := types.NamespacedName{Namespace: namespace, Name: node}
	return Eventually(func() nmstatev1alpha1.State {
		*currentStateYaml = nodeNetworkState(key).Status.CurrentState
		return *currentStateYaml
	}, ReadTimeout, ReadInterval)
}

func desiredState(namespace string, node string, desiredStateYaml *nmstatev1alpha1.State) AsyncAssertion {
	key := types.NamespacedName{Namespace: namespace, Name: node}
	return Eventually(func() nmstatev1alpha1.State {
		*desiredStateYaml = nodeNetworkState(key).Spec.DesiredState
		return *desiredStateYaml
	}, ReadTimeout, ReadInterval)
}

func interfacesNameForNode(node string) AsyncAssertion {
	return Eventually(func() []string {
		var currentStateYaml nmstatev1alpha1.State
		currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

		interfaces := interfaces(currentStateYaml)
		Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

		return interfacesName(interfaces)
	}, ReadTimeout, ReadInterval)
}

func interfacesForNode(node string) AsyncAssertion {
	return Eventually(func() []interface{} {
		var currentStateYaml nmstatev1alpha1.State
		currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

		interfaces := interfaces(currentStateYaml)
		Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

		return interfaces
	}, ReadTimeout, ReadInterval)
}

func toUnstructured(y string) interface{} {
	var u interface{}
	err := yaml.Unmarshal([]byte(y), &u)
	Expect(err).ToNot(HaveOccurred())
	return u
}
