package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"

	corev1 "k8s.io/api/core/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	f                    = framework.Global
	t                    *testing.T
	namespace            string
	nodes                []string
	startTime            time.Time
	bond1                string
	bridge1              string
	primaryNic           = flag.String("primaryNic", "eth0", "Primary network interface name")
	firstSecondaryNic    = flag.String("firstSecondaryNic", "eth1", "First secondary network interface name")
	secondSecondaryNic   = flag.String("secondSecondaryNic", "eth2", "Second secondary network interface name")
	nodesInterfacesState = make(map[string][]byte)
	interfacesToIgnore   = []string{"flannel.1", "dummy0"}
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nodeNetworkStateList := &nmstatev1alpha1.NodeNetworkStateList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	err = framework.Global.Client.List(context.TODO(), &nodeList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}
})

func TestMain(m *testing.M) {
	framework.MainEntry(m)
}

func TestE2E(tapi *testing.T) {
	t = tapi
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.functest.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "E2E Test Suite", []Reporter{junitReporter})

}

var _ = BeforeEach(func() {
	bond1 = nextBond()
	bridge1 = nextBridge()
	_, namespace = prepare(t)
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
		nodeState := nodeInterfacesState(node, interfacesToIgnore)
		Expect(nodesInterfacesState[node]).Should(MatchJSON(nodeState))
		writePodsLogs(namespace, startTime, GinkgoWriter)
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
