/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This module is meant to cover all the demos as we show them. To make it as close
// to the reality as possible, we use only kubectl direcly
package handler

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
)

var _ = Describe("[user-guide] Introduction", func() {
	runConfiguration := func() {
		kubectlAndCheck("apply", "-f", "docs/user-guide/bond0-eth1-eth2_up.yaml")
		kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "4m")
		kubectlAndCheck("apply", "-f", "docs/user-guide/bond0-eth1-eth2_absent.yaml")
		kubectlAndCheck("wait", "nncp", "bond0-eth1-eth2", "--for", "condition=Available", "--timeout", "4m")
		kubectlAndCheck("delete", "nncp", "bond0-eth1-eth2")

		kubectlAndCheck("apply", "-f", "docs/user-guide/eth1-eth2_up.yaml")
		kubectlAndCheck("wait", "nncp", "eth1", "eth2", "--for", "condition=Available", "--timeout", "4m")
		kubectlAndCheck("delete", "nncp", "eth1", "eth2")

		kubectlAndCheck("apply", "-f", "docs/user-guide/vlan100_node01_up.yaml")
		kubectlAndCheck("wait", "nncp", "vlan100", "--for", "condition=Available", "--timeout", "4m")
	}

	// Policies are not deleted as a part of the tutorial, so we need additional function here
	cleanupConfiguration := func() {
		deletePolicy("vlan100")
		setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, interfaceAbsent("eth1.100"))
		policy.WaitForAvailableTestPolicy()
		resetDesiredStateForAllNodes()
	}

	runTroubleshooting := func() {
		kubectlAndCheck("apply", "-f", "docs/user-guide/eth666_up.yaml")
		kubectlAndCheck("wait", "nncp", "eth666", "--for", "condition=Degraded", "--timeout", "4m")
		kubectlAndCheck("delete", "nncp", "eth666")
	}

	BeforeEach(func() {
		skipIfNotKubernetes()
	})

	Context("Configuration tutorial", func() {
		AfterEach(func() {
			cleanupConfiguration()
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
		})

		It("should succeed executing all the commands", func() {
			runConfiguration()
			runTroubleshooting()
		})
	})
})
