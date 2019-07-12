package nodenetworkconfigurationpolicy

import (
	"os"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeNetworkConfigurationPolicy controller Test Suite")
}

var _ = BeforeSuite(func() {
	os.Setenv("NODE_NAME", "node01")
})
