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
	"time"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LLDP configuration with nmpolicy", func() {
	lldpEnabledPolicyName := "lldp-enabled"
	lldpDisabledPolicyName := "lldp-disabled"

	configureLldpOnEthernetsCapture := func(enabled string) map[string]string {
		return map[string]string{
			"ethernets":      `interfaces.type=="ethernet"`,
			"ethernets-up":   `capture.ethernets | interfaces.state=="up"`,
			"ethernets-lldp": fmt.Sprintf(`capture.ethernets-up | interfaces.lldp.enabled:=%s`, enabled),
		}
	}

	interfacesWithLldpEnabledState := nmstate.NewState(`interfaces: "{{ capture.ethernets-lldp.interfaces }}"`)

	BeforeEach(func() {
		By("Enabling LLDP on up ethernet interfaces")
		setDesiredStateWithPolicyAndCapture(lldpEnabledPolicyName, interfacesWithLldpEnabledState, configureLldpOnEthernetsCapture("true"))
		policy.WaitForAvailablePolicy(lldpEnabledPolicyName)

		DeferCleanup(func() {
			deletePolicy(lldpEnabledPolicyName)

			By("Disabling LLDP on up ethernet interfaces")
			setDesiredStateWithPolicyAndCapture(lldpDisabledPolicyName, interfacesWithLldpEnabledState, configureLldpOnEthernetsCapture("false"))
			policy.WaitForAvailablePolicy(lldpDisabledPolicyName)
			deletePolicy(lldpDisabledPolicyName)
		})
	})

	It("should enable LLDP on all ethernet interfaces that are up", func() {
		Byf("Check %s has lldp enabled", primaryNic)
		for _, node := range nodes {
			Eventually(
				func() string {
					return lldpEnabled(node, primaryNic)
				},
				30*time.Second, 1*time.Second,
			).Should(Equal("true"), fmt.Sprintf("Interface %s should have enabled lldp", primaryNic))
		}
	})
})
