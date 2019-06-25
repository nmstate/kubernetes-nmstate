package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	f         = framework.Global
	t         *testing.T
	namespace string
	nodes     = []string{"node01"} // TODO: Get it from cluster
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nodeNetworkStateList := &nmstatev1alpha1.NodeNetworkStateList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, nodeNetworkStateList)
	Expect(err).ToNot(HaveOccurred())
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
})

var _ = AfterEach(func() {
	writePodsLogs(namespace, GinkgoWriter)
})
