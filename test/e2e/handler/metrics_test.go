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
	"strconv"
	"strings"
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
		simpleBridge = func(bridge string) nmstate.State {
			return nmstate.NewState(fmt.Sprintf(`
interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port: []
`, bridge))
		}
		staticRouteState = func(nic, ipAddress, destIPAddress, nextHopIPAddress string) nmstate.State {
			return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      address:
      - ip: %s
        prefix-length: "24"
      dhcp: false
      enabled: true
routes:
    config:
    - destination: %s
      metric: 150
      next-hop-address: %s
      next-hop-interface: %s
      table-id: 254
`, nic, ipAddress, destIPAddress, nextHopIPAddress, nic))
		}
		staticRouteAbsent = func(nic string) nmstate.State {
			return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: false
routes:
    config:
    - next-hop-interface: %s
      state: absent
`, nic, nic))
		}
	)
	Context("when desiredState is configured", func() {
		Context("with a state that increases features gauge", func() {
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
						ShouldNot(HaveKey(monitoring.AppliedFeaturesOpts.Name + `{name="dhcpv4-custom-hostname"}`))
				})
			})
		})

		Context("with interface type metric tracking", func() {
			var initialBridgeCount int

			BeforeEach(func() {
				By("Getting initial linux-bridge count before creating bridge")
				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() map[string]string {
					return getMetrics(token)
				}).WithPolling(time.Second).WithTimeout(5 * time.Second).ShouldNot(BeEmpty())
				metrics := getMetrics(token)
				initialBridgeCount = sumInterfaceTypeMetric(metrics, "linux-bridge")
			})

			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
				resetDesiredStateForNodes()
			})

			It("should increase and decrease linux-bridge count", func() {
				By("Creating a linux-bridge first")
				updateDesiredStateAndWait(simpleBridge(bridge1))

				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for bridge count to increase")
				expectedAfterCreate := initialBridgeCount + len(nodes)
				Eventually(func() int {
					metrics := getMetrics(token)
					return sumInterfaceTypeMetric(metrics, "linux-bridge")
				}).
					WithPolling(time.Second).
					WithTimeout(30 * time.Second).
					Should(Equal(expectedAfterCreate))

				By("Deleting the linux-bridge")
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))

				By("Verifying linux-bridge count decreased back to initial")
				Eventually(func() int {
					metrics := getMetrics(token)
					return sumInterfaceTypeMetric(metrics, "linux-bridge")
				}).
					WithPolling(time.Second).
					WithTimeout(30 * time.Second).
					Should(Equal(initialBridgeCount))
			})
		})

		Context("with static routes metric tracking", func() {
			var initialStaticRouteCount int

			BeforeEach(func() {
				By("Getting initial static route count before creating routes")
				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() map[string]string {
					return getMetrics(token)
				}).WithPolling(time.Second).WithTimeout(5 * time.Second).ShouldNot(BeEmpty())
				metrics := getMetrics(token)
				initialStaticRouteCount = sumRouteMetric(metrics, "ipv4", "static")
			})

			AfterEach(func() {
				updateDesiredStateAtNodeAndWait(nodes[0], staticRouteAbsent(firstSecondaryNic))
				resetDesiredStateForNodes()
			})

			It("should increase and decrease static route count", func() {
				By("Creating a static route")
				updateDesiredStateAtNodeAndWait(nodes[0], staticRouteState(firstSecondaryNic, "192.168.100.1", "192.168.200.0/24", "192.168.100.254"))

				token, err := getPrometheusToken()
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for static route count to increase")
				expectedAfterCreate := initialStaticRouteCount + 1
				Eventually(func() int {
					metrics := getMetrics(token)
					return sumRouteMetric(metrics, "ipv4", "static")
				}).
					WithPolling(time.Second).
					WithTimeout(30 * time.Second).
					Should(Equal(expectedAfterCreate))

				By("Deleting the static route")
				updateDesiredStateAtNodeAndWait(nodes[0], staticRouteAbsent(firstSecondaryNic))

				By("Verifying static route count decreased back to initial")
				Eventually(func() int {
					metrics := getMetrics(token)
					return sumRouteMetric(metrics, "ipv4", "static")
				}).
					WithPolling(time.Second).
					WithTimeout(30 * time.Second).
					Should(Equal(initialStaticRouteCount))
			})
		})
	})
})

// sumInterfaceTypeMetric sums the metric values for a given interface type across all nodes
func sumInterfaceTypeMetric(metrics map[string]string, ifaceType string) int {
	total := 0
	metricPrefix := monitoring.NetworkInterfacesOpts.Name + `{`
	for key, value := range metrics {
		if strings.HasPrefix(key, metricPrefix) && strings.Contains(key, fmt.Sprintf(`type="%s"`, ifaceType)) {
			if v, err := strconv.Atoi(value); err == nil {
				total += v
			}
		}
	}
	return total
}

// sumRouteMetric sums the metric values for routes with given ipStack and routeType across all nodes
func sumRouteMetric(metrics map[string]string, ipStack, routeType string) int {
	total := 0
	metricPrefix := monitoring.NetworkRoutesOpts.Name + `{`
	for key, value := range metrics {
		if strings.HasPrefix(key, metricPrefix) &&
			strings.Contains(key, fmt.Sprintf(`ip_stack="%s"`, ipStack)) &&
			strings.Contains(key, fmt.Sprintf(`type="%s"`, routeType)) {
			if v, err := strconv.Atoi(value); err == nil {
				total += v
			}
		}
	}
	return total
}
