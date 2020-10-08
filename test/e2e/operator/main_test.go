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

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/nmstate/kubernetes-nmstate/api"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	f         = framework.Global
	t         *testing.T
	nodes     []string
	startTime time.Time
	ctx       *framework.TestCtx
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nmstateList := &nmstatev1beta1.NMStateList{}

	err := framework.AddToFrameworkScheme(api.AddToScheme, nmstateList)
	Expect(err).ToNot(HaveOccurred())
	ctx, _ = prepare(t)
})

var _ = AfterSuite(func() {
	ctx.Cleanup()
})

func TestMain(m *testing.M) {
	framework.MainEntry(m)
}

func TestE2E(tapi *testing.T) {
	t = tapi
	RegisterFailHandler(Fail)

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	err := framework.Global.Client.List(context.TODO(), &nodeList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}

	reporters := make([]Reporter, 0)
	reporters = append(reporters, knmstatereporter.New("test_logs/e2e/operator", framework.Global.OperatorNamespace, nodes))
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
