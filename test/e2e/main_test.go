package e2e

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	f         = framework.Global
	t         *testing.T
	namespace string
	nodes     []string
	startTime time.Time
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
	_, namespace = prepare(t)
	startTime = time.Now()
})

var _ = AfterEach(func() {
	writePodsLogs(namespace, startTime, GinkgoWriter)
})
