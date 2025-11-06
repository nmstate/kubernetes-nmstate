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
			vlanIDs    = []string{"102", "103"}
			policyName = func(vlanID string) string {
				return "vlan-" + vlanID
			}
		)

		BeforeEach(func() {
			for _, vlanID := range vlanIDs {
				setDesiredStateWithPolicy(policyName(vlanID), ifaceUpWithVlanUp(firstSecondaryNic, vlanID))
				policy.WaitForAvailablePolicy(policyName(vlanID))
			}
		})

		AfterEach(func() {
			for _, vlanID := range vlanIDs {
				setDesiredStateWithPolicy(policyName(vlanID), vlanAbsent(firstSecondaryNic, vlanID))
				policy.WaitForAvailablePolicy(policyName(vlanID))
				deletePolicy(policyName(vlanID))
			}
			resetDesiredStateForNodes()
		})

		It("should have the vlan interfaces configured", func() {
			for _, node := range nodes {
				for _, vlanID := range vlanIDs {
					interfacesNameForNodeEventually(node).Should(ContainElement(fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanID)))
					vlanForNodeInterfaceEventually(node, fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanID)).Should(Equal(vlanID))
				}
			}
		})
	})
})
