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
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/nmstate/kubernetes-nmstate/api"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	f                    = framework.Global
	t                    *testing.T
	nodes                []string
	startTime            time.Time
	bond1                string
	bridge1              string
	primaryNic           string
	firstSecondaryNic    string
	secondSecondaryNic   string
	nodesInterfacesState = make(map[string][]byte)
	interfacesToIgnore   = []string{"flannel.1", "dummy0"}
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nodeNetworkStateList := &nmstatev1beta1.NodeNetworkStateList{}
	err := framework.AddToFrameworkScheme(api.AddToScheme, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())

	prepare(t)

	resetDesiredStateForNodes()
})

func TestMain(m *testing.M) {
	primaryNic = environment.GetVarWithDefault("PRIMARY_NIC", "eth0")
	firstSecondaryNic = environment.GetVarWithDefault("FIRST_SECONDARY_NIC", "eth1")
	secondSecondaryNic = environment.GetVarWithDefault("SECOND_SECONDARY_NIC", "eth2")
	framework.MainEntry(m)
}

func TestE2E(tapi *testing.T) {
	t = tapi
	RegisterFailHandler(Fail)

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	filterWorkers := dynclient.MatchingLabels{"node-role.kubernetes.io/worker": ""}
	err := framework.Global.Client.List(context.TODO(), &nodeList, filterWorkers)
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	reporters := make([]Reporter, 0)
	reporters = append(reporters, knmstatereporter.New("test_logs/e2e/handler", framework.Global.OperatorNamespace, nodes))
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
	bridge1 = nextBridge()
	startTime = time.Now()
	By("Getting nodes initial state")
	for _, node := range nodes {
		nodeState := nodeInterfacesState(node, interfacesToIgnore)
		nodesInterfacesState[node] = nodeState
	}
})

var _ = AfterEach(func() {
	By("Verifying initial state")
	for _, node := range nodes {

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
