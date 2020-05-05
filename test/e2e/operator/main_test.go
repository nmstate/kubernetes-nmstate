package operator

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgoreporters "github.com/onsi/ginkgo/reporters"

	corev1 "k8s.io/api/core/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	f         = framework.Global
	t         *testing.T
	nodes     []string
	startTime time.Time
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nmstateList := &nmstatev1alpha1.NMStateList{}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, nmstateList)
	Expect(err).ToNot(HaveOccurred())
	prepare(t)
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
	knmstateReporter := knmstatereporter.New("test_logs/e2e/operator", framework.Global.Namespace, nodes)
	junitReporter := ginkgoreporters.NewJUnitReporter("junit.functest.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Operator E2E Test Suite", []Reporter{junitReporter, knmstateReporter})

}

var _ = BeforeEach(func() {
})

var _ = AfterEach(func() {
})
