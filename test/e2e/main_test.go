package e2e

import (
	"context"
	"flag"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	f                  = framework.Global
	t                  *testing.T
	namespace          string
	nodes              []string
	startTime          time.Time
	bond1              string
	bridge1            string
	primaryNic         = flag.String("primaryNic", "eth0", "Primary network interface name")
	firstSecondaryNic  = flag.String("firstSecondaryNic", "eth1", "First secondary network interface name")
	secondSecondaryNic = flag.String("secondSecondaryNic", "eth2", "Second secondary network interface name")
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nodeNetworkStateList := &nmstatev1alpha1.NodeNetworkStateList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	err = framework.Global.Client.List(context.TODO(), &dynclient.ListOptions{}, &nodeList)
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
	RunSpecs(t, "E2E Test Suite")
}

var _ = BeforeEach(func() {
	bond1 = nextBond()
	bridge1 = nextBridge()
	_, namespace = prepare(t)
	startTime = time.Now()
})

var _ = AfterEach(func() {
	writePodsLogs(namespace, startTime, GinkgoWriter)
})
