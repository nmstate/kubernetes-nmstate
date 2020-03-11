// This module is meant to cover all the demos as we show them. To make it as close
// to the reality as possible, we use only kubectl direcly
package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo"
)

func skipIfNotKubernetes() {
	provider := getEnv("KUBEVIRT_PROVIDER", "k8s")
	if !strings.Contains(provider, "k8s") {
		Skip("Tutorials use interface naming that is available only on Kubernetes providers")
	}
}

var _ = Describe("Introduction", func() {
	runConfiguration := func() {
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
	}

	// Policies are not deleted as a part of the tutorial, so we need additional function here
	cleanupConfiguration := func() {
		deletePolicy("vlan100")
		updateDesiredState(interfaceAbsent("eth1.100"))
		waitForAvailableTestPolicy()
	}

	runTroubleshooting := func() {
		kubectlAndCheck("apply", "-f", "docs/examples/eth666_up.yaml")
		kubectlAndCheck("wait", "nncp", "eth666", "--for", "condition=Degraded", "--timeout", "2m")
		kubectlAndCheck("delete", "nncp", "eth666")
	}

	BeforeEach(func() {
		skipIfNotKubernetes()
	})

	Context("Configuration tutorial", func() {
		AfterEach(func() {
			cleanupConfiguration()
			resetDesiredStateForNodes()
		})

		It("should succeed executing all the commands", func() {
			runConfiguration()
		})
	})

	Context("Troubleshooting tutorial", func() {
		It("should succeed executing all the commands", func() {
			runTroubleshooting()
		})
	})

	Context("All tutorials in a row", func() {
		AfterEach(func() {
			cleanupConfiguration()
			resetDesiredStateForNodes()
		})

		It("should succeed executing all the commands", func() {
			runConfiguration()
			runTroubleshooting()
		})
	})
})
