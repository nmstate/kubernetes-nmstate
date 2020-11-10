package operator

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgoreporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	corev1 "k8s.io/api/core/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	t         *testing.T
	nodes     []string
	startTime time.Time
)

var _ = BeforeSuite(func() {

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testenv.Start()

})

func TestMain(m *testing.M) {
	testenv.TestMain()
}

func TestE2E(tapi *testing.T) {
	t = tapi
	RegisterFailHandler(Fail)

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	err := testenv.Client.List(context.TODO(), &nodeList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	reporters := make([]Reporter, 0)
	reporters = append(reporters, knmstatereporter.New("test_logs/e2e/operator", testenv.OperatorNamespace, nodes))
	if ginkgoreporters.Polarion.Run {
		reporters = append(reporters, &ginkgoreporters.Polarion)
	}
	if ginkgoreporters.JunitOutput != "" {
		reporters = append(reporters, ginkgoreporters.NewJunitReporter())
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Operator E2E Test Suite", reporters)
}

var _ = BeforeEach(func() {
})

var _ = AfterEach(func() {
})
