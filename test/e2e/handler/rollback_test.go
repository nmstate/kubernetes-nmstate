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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
)

// We cannot change routes at nmstate if the interface is with dhcp true
// that's why we need to set it static with the same ip it has previously.
func badDefaultGw(address, nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      dhcp: false
      enabled: true
      address:
        - ip: %s
          prefix-length: 24
routes:
  config:
    - destination: 0.0.0.0/0
      metric: 150
      next-hop-address: 192.0.2.1
      next-hop-interface: %s
`, nic, address, nic))
}

func removeNameServers(nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    search: []
    server: []
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: false
    dhcp: true
    enabled: true
  ipv6:
    auto-dns: false
    dhcp: true
    enabled: true
`, nic))
}

func setBadNameServers(nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    search: []
    server:
      - 192.168.100.3
      - 192.168.100.4
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: false
    dhcp: true
    enabled: true
  ipv6:
    auto-dns: false
    dhcp: true
    enabled: true
`, nic))
}

func discoverNameServers(nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: true
    dhcp: true
    enabled: true
  ipv6:
    auto-dns: true
    dhcp: true
    enabled: true
`, nic))
}

var _ = Describe("rollback", func() {
	// This spec is done only at first node since policy has to be different
	// per node (ip addresses has to be different at cluster).
	Context("when connectivity to default gw is lost after state configuration", func() {
		BeforeEach(func() {
			By("Configure a invalid default gw")
			var address string
			Eventually(func() string {
				address = ipv4Address(nodes[0], primaryNic)
				return address
			}, ReadTimeout, ReadInterval).ShouldNot(BeEmpty())
			updateDesiredStateAtNode(nodes[0], badDefaultGw(address, primaryNic))
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to a good gw configuration", func() {
			By("Should not be available") // Fail fast
			policy.StatusConsistently().ShouldNot(policy.ContainPolicyAvailable())

			By("Wait for reconcile to fail")
			policy.WaitForDegradedTestPolicy()
			Byf("Check that %s is rolled back", primaryNic)
			Eventually(func() bool {
				return dhcpFlag(nodes[0], primaryNic)
			}, 480*time.Second, ReadInterval).Should(BeTrue(), "DHCP flag hasn't rollback to true")

			Byf("Check that %s continue with rolled back state", primaryNic)
			Consistently(func() bool {
				return dhcpFlag(nodes[0], primaryNic)
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "DHCP flag has change to false")
		})
	})

	Context("when name servers are lost after state configuration", func() {
		BeforeEach(func() {
			updateDesiredStateAtNode(nodes[0], removeNameServers(primaryNic))
		})
		AfterEach(func() {
			updateDesiredStateAtNode(nodes[0], discoverNameServers(primaryNic))
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to previous name servers", func() {
			By("Should not be available") // Fail fast
			policy.StatusConsistently().ShouldNot(policy.ContainPolicyAvailable())

			By("Wait for reconcile to fail")
			policy.WaitForDegradedTestPolicy()
			Byf("Check that %s is rolled back", primaryNic)
			Eventually(func() bool {
				return autoDNS(nodes[0], primaryNic)
			}, 480*time.Second, ReadInterval).Should(BeTrue(), "should eventually have auto-dns=true")

			Byf("Check that %s continue with rolled back state", primaryNic)
			Consistently(func() bool {
				return autoDNS(nodes[0], primaryNic)
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "should consistently have auto-dns=true")

		})
	})

	Context("when name servers are wrong after state configuration", func() {
		BeforeEach(func() {
			updateDesiredStateAtNode(nodes[0], setBadNameServers(primaryNic))
		})
		AfterEach(func() {
			updateDesiredStateAtNode(nodes[0], discoverNameServers(primaryNic))
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to previous name servers", func() {
			By("Should not be available") // Fail fast
			policy.StatusConsistently().ShouldNot(policy.ContainPolicyAvailable())

			By("Wait for reconcile to fail")
			policy.WaitForDegradedTestPolicy()
			Byf("Check that %s is rolled back", primaryNic)
			Eventually(func() bool {
				return autoDNS(nodes[0], primaryNic)
			}, 480*time.Second, ReadInterval).Should(BeTrue(), "should eventually have auto-dns=true")

			Byf("Check that %s continue with rolled back state", primaryNic)
			Consistently(func() bool {
				return autoDNS(nodes[0], primaryNic)
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "should consistently have auto-dns=true")

		})
	})

})
