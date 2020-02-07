// This module is meant to cover all the demos as we show them. To make it as close
// to the reality as possible, we use only kubectl direcly
package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/test/kubectl"
)

func kubectlAndCheck(command ...string) {
	out, err := kubectl.Kubectl(command...)
	Expect(err).ShouldNot(HaveOccurred(), out)
}

var _ = Describe("Introduction: Configuration", func() {
	Context("following the tutorial", func() {
		FIt("should succeed executing all the commands", func() {
			for i := 0; i < 50; i++ {
				kubectlAndCheck("apply", "-f", "docs/examples/bond0-eth1-eth2_up.yaml")
				kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
				kubectlAndCheck("apply", "-f", "docs/examples/bond0-eth1-eth2_absent.yaml")
				kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
				kubectlAndCheck("delete", "nncp", "bond0-eth1-eth2")
				kubectlAndCheck("apply", "-f", "docs/examples/eth1-eth2_up.yaml")
				kubectlAndCheck("wait", "nncp", "eth1", "eth2", "--for", "condition=Available", "--timeout", "2m")
				kubectlAndCheck("delete", "nncp", "eth1", "eth2")
			}
		})
	})
})
