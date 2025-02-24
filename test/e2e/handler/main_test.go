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

package handler

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	allNodes                    []string
	nodes                       []string
	startTime                   time.Time
	bond1                       string
	bridge1                     string
	primaryNic                  string
	firstSecondaryNic           string
	secondSecondaryNic          string
	dnsTestNic                  string
	portFieldName               string
	miimonFormat                string
	nodesInitialInterfacesState = make(map[string]map[string]string)
	interfacesToIgnore          = []string{"flannel.1", "dummy0", "tunl0"}
	knmstateReporter            *knmstatereporter.KubernetesNMStateReporter
)

var _ = BeforeSuite(func() {

	// Change to root directory some test expect that
	os.Chdir("../../../")

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	primaryNic = environment.GetVarWithDefault("PRIMARY_NIC", "eth0")
	firstSecondaryNic = environment.GetVarWithDefault("FIRST_SECONDARY_NIC", "eth1")
	secondSecondaryNic = environment.GetVarWithDefault("SECOND_SECONDARY_NIC", "eth2")
	dnsTestNic = environment.GetVarWithDefault("DNS_TEST_NIC", primaryNic)

	testenv.Start()

	portFieldName = "port"
	miimonFormat = "%d"

	By("Getting nmstate-enabled node list from cluster")
	podList := corev1.PodList{}
	filterHandlers := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
	err := testenv.Client.List(context.TODO(), &podList, filterHandlers)
	Expect(err).ToNot(HaveOccurred())
	for _, pod := range podList.Items {
		allNodes = append(allNodes, pod.Spec.NodeName)
	}

	By("Getting nmstate-enabled worker node list from cluster")
	nodeList := corev1.NodeList{}
	filterWorkers := client.MatchingLabels{"node-role.kubernetes.io/worker": ""}
	err = testenv.Client.List(context.TODO(), &nodeList, filterWorkers)
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		if containsNode(allNodes, node.Name) {
			nodes = append(nodes, node.Name)
		}
	}

	resetDesiredStateForAllNodes()
	expectedInitialState := interfacesState(resetPrimaryAndSecondaryNICs(), interfacesToIgnore)
	for _, node := range allNodes {
		Eventually(func(g Gomega) {
			By("Wait for network configuration to show up at NNS to retrieve it")
			nodeState := nodeInterfacesState(node, interfacesToIgnore)
			for name, state := range expectedInitialState {
				g.Expect(nodeState).To(HaveKeyWithValue(name, state))
			}
		}, 20*time.Second, time.Second).Should(Succeed())
	}
	knmstateReporter = knmstatereporter.New("test_logs/e2e/handler", testenv.OperatorNamespace, nodes)
	knmstateReporter.Cleanup()
	By("Getting nodes initial state")
	for _, node := range allNodes {
		nodeState := nodeInterfacesState(node, interfacesToIgnore)
		nodesInitialInterfacesState[node] = nodeState
	}
})

func TestE2E(t *testing.T) {
	testenv.TestMain()

	RegisterFailHandler(Fail)

	RunSpecs(t, "Handler E2E Test Suite")
}

var _ = BeforeEach(func() {
	bond1 = nextBond()
	Byf("Setting bond1=%s", bond1)
	bridge1 = nextBridge()
	Byf("Setting bridge1=%s", bridge1)
	startTime = time.Now()
})

var _ = AfterEach(func() {
	By("Verifying initial state")
	for _, node := range allNodes {
		Eventually(func() map[string]string {
			By("Verifying initial state eventually")
			nodeState := nodeInterfacesState(node, interfacesToIgnore)
			return nodeState
		}, 120*time.Second, 5*time.Second).Should(Equal(nodesInitialInterfacesState[node]), fmt.Sprintf("Test didn't return "+
			"to initial state on node %s", node))
	}
})

var _ = ReportBeforeEach(func(specReport types.SpecReport) {
	knmstateReporter.ReportBeforeEach(specReport)
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	knmstateReporter.ReportAfterEach(specReport)
})

func containsNode(nodes []string, node string) bool {
	for _, n := range nodes {
		if n == node {
			return true
		}
	}
	return false
}
