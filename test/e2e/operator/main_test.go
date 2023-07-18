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

package operator

import (
	"context"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

const manifestsDir = "build/_output/manifests/"

var (
	nodes            []string
	knmstateReporter *knmstatereporter.KubernetesNMStateReporter
	manifestFiles    = []string{"namespace.yaml", "service_account.yaml", "operator.yaml", "role.yaml", "role_binding.yaml"}
	defaultOperator  TestData
)

func TestE2E(t *testing.T) {
	testenv.TestMain()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator E2E Test Suite")
}

var _ = BeforeSuite(func() {
	// Change to root directory some test expect that
	os.Chdir("../../../")

	defaultOperator = NewOperatorTestData(os.Getenv("HANDLER_NAMESPACE"), manifestsDir, manifestFiles)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testenv.Start()

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	Expect(testenv.Client.List(context.TODO(), &nodeList, &client.ListOptions{})).To(Succeed())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	knmstateReporter = knmstatereporter.New("test_logs/e2e/operator", testenv.OperatorNamespace, nodes)
	knmstateReporter.Cleanup()
})

var _ = AfterSuite(func() {
	UninstallNMStateAndWaitForDeletion(defaultOperator)
})

var _ = ReportBeforeEach(func(specReport ginkgotypes.SpecReport) {
	knmstateReporter.ReportBeforeEach(specReport)
})

var _ = ReportAfterEach(func(specReport ginkgotypes.SpecReport) {
	// XXX(mko) This is a hack because in OCP CI we notice that AfterEach reporter is hanging
	// indefinitely. We are simply ignoring it so that tests can proceed. An ultimate solution
	// would be to implement a proper WithTimeout context which runs "runAndWait" function in the
	// test/reporter/reporter.go
	if !isKubevirtciCluster() {
		return
	}
	knmstateReporter.ReportAfterEach(specReport)
})

func podsShouldBeDistributedAtNodes(selectedNodes []corev1.Node, listOptions ...client.ListOption) {
	podList := &corev1.PodList{}
	Expect(testenv.Client.List(context.TODO(), podList, listOptions...)).To(Succeed())
	nodesRunningPod := map[string]bool{}
	for _, pod := range podList.Items {
		Expect(pod.Spec.NodeName).To(BeElementOf(namesFromNodes(selectedNodes)), "should run on the selected nodes")
		nodesRunningPod[pod.Spec.NodeName] = true
	}
	if len(selectedNodes) > 1 {
		Expect(nodesRunningPod).To(HaveLen(len(podList.Items)), "should run pods at different nodes")
	} else {
		Expect(nodesRunningPod).To(HaveLen(1), "should run pods at the same node")
	}
}

func isKubevirtciCluster() bool {
	return strings.Contains(os.Getenv("KUBECONFIG"), "kubevirtci")
}

func controlPlaneNodes() []corev1.Node {
	nodeList := &corev1.NodeList{}
	Expect(testenv.Client.List(context.TODO(), nodeList, client.HasLabels{"node-role.kubernetes.io/control-plane"})).To(Succeed())
	if len(nodeList.Items) == 0 {
		Expect(testenv.Client.List(context.TODO(), nodeList, client.HasLabels{"node-role.kubernetes.io/master"})).To(Succeed())
	}
	return nodeList.Items
}

func namesFromNodes(nodes []corev1.Node) []string {
	names := []string{}
	for _, node := range nodes {
		names = append(names, node.Name)
	}
	return names
}
