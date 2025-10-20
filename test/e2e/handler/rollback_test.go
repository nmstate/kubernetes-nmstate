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
func badDefaultGw(address, nic string, routingTable int) nmstate.State {
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
      table-id: %d
`, nic, address, nic, routingTable))
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
    autoconf: true
    enabled: true
`, nic))
}

func setBadNameServers(nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    search: []
    server:
      - "fe80::deef:1%%%[1]s"
      - "fe80::deef:2%%%[1]s"
interfaces:
- name: %[1]s
  type: ethernet
  state: up
  ipv4:
    auto-dns: false
    dhcp: true
    enabled: true
  ipv6:
    auto-dns: false
    dhcp: true
    autoconf: true
    enabled: true
`, nic))
}

func setBadNameServersGlobal(addr string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    search: []
    server:
      - %s
`, addr))
}

func discoverNameServers(nic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`
dns-resolver:
  config:
    search: []
    server: []
interfaces:
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
    autoconf: true
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
			updateDesiredStateAtNode(nodes[0], badDefaultGw(address, primaryNic, 254))
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
			}, 600*time.Second, ReadInterval).Should(BeTrue(), "DHCP flag hasn't rollback to true")

			Byf("Check that %s continue with rolled back state", primaryNic)
			Consistently(func() bool {
				return dhcpFlag(nodes[0], primaryNic)
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "DHCP flag has change to false")
		})
	})

	Context("when changing the default gw to a routing table different from main", func() {
		secondaryNicCustomAddress := "192.168.100.1"
		BeforeEach(func() {
			By("Configure a invalid default gw")
			applyTime := time.Now()
			updateDesiredStateAtNode(nodes[0], badDefaultGw(secondaryNicCustomAddress, firstSecondaryNic, 200))
			policy.WaitForPolicyTransitionUpdateWithTime(TestPolicy, applyTime)
			policy.WaitForAvailablePolicy(TestPolicy)
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should not rollback to the previous configuration", func() {
			Eventually(func() string {
				return ipv4Address(nodes[0], firstSecondaryNic)
			}, 3*time.Minute, ReadInterval).Should(Equal(secondaryNicCustomAddress), "IP has not being set")

			Byf("Check that %s is not rolled back", firstSecondaryNic)
			Consistently(func() string {
				return ipv4Address(nodes[0], firstSecondaryNic)
			}, 20*time.Second, ReadInterval).Should(Equal(secondaryNicCustomAddress), "IP has rolled back to empty")
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

	Context("when name servers (configured globally) are wrong after state configuration", func() {
		BeforeEach(func() {
			updateDesiredStateAtNode(nodes[0], setBadNameServersGlobal("192.168.100.3"))
		})
		AfterEach(func() {
			By("Clean up desired state")
			resetDesiredStateForNodes()
		})
		It("should rollback to previous name servers", func() {
			By("Should not be available") // Fail fast
			policy.StatusConsistently().ShouldNot(policy.ContainPolicyAvailable())

			By("Wait for reconcile to fail")
			policy.WaitForDegradedTestPolicy()
			By("Check that global DNS is rolled back")
			Eventually(func() []string {
				return dnsResolverForNode(nodes[0], "dns-resolver.running.server")
			}, 600*time.Second, ReadInterval).ShouldNot(ContainElement("192.168.100.3"), "should eventually lose wrong name server")

			By("Check that global DNS continue with rolled back state")
			Consistently(func() []string {
				return dnsResolverForNode(nodes[0], "dns-resolver.running.server")
			}, 5*time.Second, 1*time.Second).ShouldNot(ContainElement("192.168.100.3"), "should consistently not contain wrong name server")
		})
	})
})
