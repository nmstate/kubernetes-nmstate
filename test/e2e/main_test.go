package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apis "github.com/nmstate/kubernetes-nmstate/pkg/apis"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var (
	f = framework.Global
	t *testing.T
)

var _ = BeforeSuite(func() {
	By("Adding custom resource scheme to framework")
	nodeNetworkStateList := &nmstatev1.NodeNetworkStateList{}
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
