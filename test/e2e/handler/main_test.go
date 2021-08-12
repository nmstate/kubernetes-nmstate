package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgoreporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	corev1 "k8s.io/api/core/v1"

	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

var (
	t                    *testing.T
	allNodes             []string
	nodes                []string
	startTime            time.Time
	bond1                string
	bridge1              string
	primaryNic           string
	firstSecondaryNic    string
	secondSecondaryNic   string
	portFieldName        string
	miimonFormat         string
	nodesInterfacesState = make(map[string][]byte)
	interfacesToIgnore   = []string{"flannel.1", "dummy0"}
)

var _ = BeforeSuite(func() {

	// Change to root directory some test expect that
	os.Chdir("../../../")

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	primaryNic = environment.GetVarWithDefault("PRIMARY_NIC", "eth0")
	firstSecondaryNic = environment.GetVarWithDefault("FIRST_SECONDARY_NIC", "eth1")
	secondSecondaryNic = environment.GetVarWithDefault("SECOND_SECONDARY_NIC", "eth2")

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
	for _, node := range nodeList.Items {
		if containsNode(allNodes, node.Name) {
			nodes = append(nodes, node.Name)
		}
	}

	resetDesiredStateForNodes()
})

func TestE2E(t *testing.T) {
	testenv.TestMain()

	RegisterFailHandler(Fail)

	reporters := make([]Reporter, 0)
	reporters = append(reporters, knmstatereporter.New("test_logs/e2e/handler", testenv.OperatorNamespace, nodes))
	if ginkgoreporters.Polarion.Run {
		reporters = append(reporters, &ginkgoreporters.Polarion)
	}
	if ginkgoreporters.JunitOutput != "" {
		reporters = append(reporters, ginkgoreporters.NewJunitReporter())
	}

	RunSpecsWithDefaultAndCustomReporters(t, "E2E Test Suite", reporters)

}

var _ = BeforeEach(func() {
	bond1 = nextBond()
	By(fmt.Sprintf("Setting bond1=%s", bond1))
	bridge1 = nextBridge()
	By(fmt.Sprintf("Setting bridge1=%s", bridge1))
	startTime = time.Now()

	By("Getting nodes initial state")
	for _, node := range allNodes {
		nodeState := nodeInterfacesState(node, interfacesToIgnore)
		nodesInterfacesState[node] = nodeState
	}
})

var _ = AfterEach(func() {
	By("Verifying initial state")
	for _, node := range allNodes {
		Eventually(func() []byte {
			By("Verifying initial state eventually")
			nodeState := nodeInterfacesState(node, interfacesToIgnore)
			return nodeState
		}, 120*time.Second, 5*time.Second).Should(MatchJSON(nodesInterfacesState[node]), fmt.Sprintf("Test didn't return "+
			"to initial state on node %s", node))
	}
})

func getMaxFailsFromEnv() int {
	maxFailsEnv := os.Getenv("REPORTER_MAX_FAILS")
	if maxFailsEnv == "" {
		return 10
	}

	maxFails, err := strconv.Atoi(maxFailsEnv)
	if err != nil { // if the variable is set with a non int value
		fmt.Println("Invalid REPORTER_MAX_FAILS variable, defaulting to 10")
		return 10
	}

	return maxFails
}

func containsNode(nodes []string, node string) bool {
	for _, n := range nodes {
		if n == node {
			return true
		}
	}
	return false
}
