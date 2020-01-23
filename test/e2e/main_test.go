package e2e

import (
	"context"
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
	primaryNic           string
	firstSecondaryNic    string
	secondSecondaryNic   string
	nodesInterfacesState = make(map[string][]byte)
	interfacesToIgnore   = []string{"flannel.1", "dummy0"}
	reporter             = NewKubernetesNMStateReporter("test_logs/e2e")
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
	reporter.BeforeSuiteDidRun()
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}
})

func TestMain(m *testing.M) {
	primaryNic = getEnv("PRIMARY_NIC", "eth0")
	firstSecondaryNic = getEnv("FIRST_SECONDARY_NIC", "eth1")
	secondSecondaryNic = getEnv("SECOND_SECONDARY_NIC", "eth2")
	framework.MainEntry(m)
}

func getEnv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		value = defaultValue
	}
	return value
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
	reporter.dumpStateBeforeEach(getTestName())

})

var _ = AfterEach(func() {
	By("Verifying initial state")
	for _, node := range nodes {
		nodeState := nodeInterfacesState(node, interfacesToIgnore)
		Expect(nodesInterfacesState[node]).Should(MatchJSON(nodeState))
	}
	reporter.dumpStateAfterEach(getTestName(), namespace, startTime)
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
