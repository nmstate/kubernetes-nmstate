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

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	var linuxBridgeWithCustomHostname = func(bridge string) nmstate.State {
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
`, bridge1))
	}
	Context("when desiredState is configured", func() {
		Context("with a state that generate metrics", func() {
			BeforeEach(func() {

				updateDesiredStateAndWait(linuxBridgeWithCustomHostname(bridge1))
			})
			AfterEach(func() {

				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				resetDesiredStateForNodes()
			})
			It("should report a metrics with proper counter increased", func() {

				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())
				metrics := getMetrics(token)
				Expect(findMetric(metrics, monitoring.ApplyTopologyTotalOpts.Name+`{name="auto_ip4 -> linux-bridge"}`)).ToNot(BeEmpty(), metrics)
				Expect(findMetric(metrics, monitoring.ApplyFeaturesTotalOpts.Name+`{name="dhcpv4-custom-hostname"}`)).ToNot(BeEmpty(), metrics)
			})
		})
	})
})
