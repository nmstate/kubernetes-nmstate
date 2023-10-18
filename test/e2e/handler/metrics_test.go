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
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	var (
		extraBridgeName               = func() string { return bridge1 + "-extra" }
		linuxBridgeWithCustomHostname = func(bridge string) nmstate.State {
			return nmstate.NewState(fmt.Sprintf(`
interfaces:
  - name: %s
    type: linux-bridge
    state: up
    ipv4:
      enabled: true
      dhcp: true
      dhcp-custom-hostname: foo
    bridge:
      options:
        stp:
          enabled: false
      port: []
`, bridge))
		}
	)
	Context("when desiredState is configured", func() {
		Context("with a state that increase gauge", func() {
			BeforeEach(func() {
				By("Apply first NNCP")
				updateDesiredStateAndWait(linuxBridgeWithCustomHostname(bridge1))

				By("Apply second NNCP with same features")
				setDesiredStateWithPolicyAndCapture(extraBridgeName(), linuxBridgeWithCustomHostname(extraBridgeName()), map[string]string{})
				policy.WaitForAvailablePolicy(extraBridgeName())
			})
			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				setDesiredStateWithPolicyAndCapture(extraBridgeName(), linuxBrAbsent(extraBridgeName()), map[string]string{})
				policy.WaitForAvailablePolicy(extraBridgeName())
				resetDesiredStateForNodes()
			})
			It("should report a metrics with proper gauge increased", func() {

				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() map[string]string {
					return getMetrics(token)
				}).
					WithPolling(time.Second).
					WithTimeout(2 * time.Second).
					Should(HaveKeyWithValue(monitoring.AppliedFeaturesOpts.Name+`{name="dhcpv4-custom-hostname"}`, "1"))
			})
			Context("and update with an state that decrease the gaugue", func() {
				BeforeEach(func() {
					updateDesiredStateAndWait(linuxBrAbsent(bridge1))
					setDesiredStateWithPolicyAndCapture(extraBridgeName(), linuxBrAbsent(extraBridgeName()), map[string]string{})
					policy.WaitForAvailablePolicy(extraBridgeName())
				})
				It("should report a metrics with proper gauge decrease", func() {
					token, err := getPrometheusToken()
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() map[string]string {
						return getMetrics(token)
					}).
						WithPolling(time.Second).
						WithTimeout(2 * time.Second).
						Should(HaveKeyWithValue(monitoring.AppliedFeaturesOpts.Name+`{name="dhcpv4-custom-hostname"}`, "0"))
				})
			})
		})
	})
})
