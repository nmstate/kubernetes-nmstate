// This module is meant to cover all the demos as we show them. To make it as close
// to the reality as possible, we use only kubectl direcly
package handler

import (
	. "github.com/onsi/ginkgo"
)

var _ = Describe("[user-guide] Introduction", func() {
	runConfiguration := func() {
		kubectlAndCheck("apply", "-f", "docs/user-guide/bond0-eth1-eth2_up.yaml")
		kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
		kubectlAndCheck("apply", "-f", "docs/user-guide/bond0-eth1-eth2_absent.yaml")
		kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "2m")
		kubectlAndCheck("delete", "nncp", "bond0-eth1-eth2")

		kubectlAndCheck("apply", "-f", "docs/user-guide/eth1-eth2_up.yaml")
		kubectlAndCheck("wait", "nncp", "eth1", "eth2", "--for", "condition=Available", "--timeout", "2m")
		kubectlAndCheck("delete", "nncp", "eth1", "eth2")

		kubectlAndCheck("apply", "-f", "docs/user-guide/vlan100_node01_up.yaml")
		kubectlAndCheck("wait", "nncp", "vlan100", "--for", "condition=Available", "--timeout", "2m")
	}

	// Policies are not deleted as a part of the tutorial, so we need additional function here
	cleanupConfiguration := func() {
		deletePolicy("vlan100")
		updateDesiredStateAndWait(interfaceAbsent("eth1.100"))
	}

	runTroubleshooting := func() {
		kubectlAndCheck("apply", "-f", "docs/user-guide/eth666_up.yaml")
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
