// This module is meant to cover all the demos as we show them. To make it as close
// to the reality as possible, we use only kubectl direcly
package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
)

func kubectlAndCheck(command ...string) {
	out, err := cmd.Kubectl(command...)
	Expect(err).ShouldNot(HaveOccurred(), out)
}

func skipIfNotKubernetes() {
	provider := getEnv("KUBEVIRT_PROVIDER", "k8s")
	if !strings.Contains(provider, "k8s") {
		Skip("Tutorials use interface naming that is available only on Kubernetes providers")
	}
}

var _ = Describe("Introduction: Configuration", func() {
	AfterEach(func() {
		skipIfNotKubernetes()
		updateDesiredState(interfaceAbsent("eth1.100"))
		waitForAvailableTestPolicy()
		resetDesiredStateForNodes()
	})

	Context("following the tutorial", func() {
		It("should succeed executing all the commands", func() {
			kubectlAndCheck("apply", "-f", "docs/examples/bond0-eth1-eth2_up.yaml")
			kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
			kubectlAndCheck("apply", "-f", "docs/examples/bond0-eth1-eth2_absent.yaml")
			kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
			kubectlAndCheck("delete", "nncp", "bond0-eth1-eth2")
			kubectlAndCheck("apply", "-f", "docs/examples/eth1-eth2_up.yaml")
			kubectlAndCheck("wait", "nncp", "eth1", "eth2", "--for", "condition=Available", "--timeout", "2m")
			kubectlAndCheck("delete", "nncp", "eth1", "eth2")
			kubectlAndCheck("apply", "-f", "docs/examples/vlan100_node01_up.yaml")
			kubectlAndCheck("wait", "nncp", "vlan100", "--for", "condition=Available", "--timeout", "2m")
		})
	})
})
