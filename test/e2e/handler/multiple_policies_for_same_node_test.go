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

package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
)

var _ = Describe("NodeNetworkState", func() {
	Context("with multiple policies configured", func() {
		var (
			staticIPPolicy = "static-ip-policy"
			vlanPolicy     = "vlan-policy"
			ipAddress      = "10.244.0.1"
			vlanID         = "102"
			prefixLen      = "24"
		)

		BeforeEach(func() {
			setDesiredStateWithPolicy(staticIPPolicy, ifaceUpWithStaticIP(firstSecondaryNic, ipAddress, prefixLen))
			policy.WaitForAvailablePolicy(staticIPPolicy)
			setDesiredStateWithPolicy(vlanPolicy, ifaceUpWithVlanUp(firstSecondaryNic, vlanID))
			policy.WaitForAvailablePolicy(vlanPolicy)
		})

		AfterEach(func() {
			setDesiredStateWithPolicy(staticIPPolicy, ifaceIPDisabled(firstSecondaryNic))
			policy.WaitForAvailablePolicy(staticIPPolicy)
			setDesiredStateWithPolicy(vlanPolicy, vlanAbsent(firstSecondaryNic, vlanID))
			policy.WaitForAvailablePolicy(vlanPolicy)
			deletePolicy(staticIPPolicy)
			deletePolicy(vlanPolicy)
			resetDesiredStateForNodes()
		})

		It("should have the IP and vlan interface configured", func() {
			for _, node := range nodes {
				ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddress))
				interfacesNameForNodeEventually(node).Should(ContainElement(fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanID)))
				vlanForNodeInterfaceEventually(node, fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanID)).Should(Equal(vlanID))
			}
		})
	})
})
