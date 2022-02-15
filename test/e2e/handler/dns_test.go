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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func dhcpAwareDNSConfig(searchDomain1, searchDomain2, server1, server2, dnsTestNic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
   search:
   - %s
   - %s
   server:
   - %s
   - %s
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: false
    enabled: true
    dhcp: true
  ipv6:
    enabled: true
    dhcp: true
    auto-dns: false
`, searchDomain1, searchDomain2, server1, server2, dnsTestNic))
}

func staticIPAndGwConfig(searchDomain1, searchDomain2, server1, server2, dnsTestNic, dnsTestNicIP string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`
dns-resolver:
  config:
   search:
   - %s
   - %s
   server:
   - %s
   - %s
routes:
  config:
  - destination: 0.0.0.0/0
    metric: 100
    next-hop-address: %s
    next-hop-interface: %s
    table-id: 253
  - destination: 0::/0
    metric: 100
    next-hop-address: fd80:0:0:0::1
    next-hop-interface: eth1
    table-id: 253
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    address:
    - ip: %s
      prefix-length: 24
    enabled: true
    dhcp: false
    auto-dns: false
- name: eth1
  type: ethernet
  state: up
  ipv6:
    enabled: true
    dhcp: false
    auto-dns: false
    address:
    - ip: 2001:db8::1:1
      prefix-length: 64
`, searchDomain1, searchDomain2, server1, server2, dnsTestNicIP, dnsTestNic, dnsTestNic, dnsTestNicIP))
}

func staticIPAbsentAndRoutesAbsent(dnsTestNic, dnsTestNicIP string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    server: []
    search: []
routes:
  config:
  - destination: 0.0.0.0/0
    metric: 100
    state: absent
    next-hop-address: %s
    next-hop-interface: %s
    table-id: 253
  - destination: 0::/0
    state: absent
    metric: 100
    next-hop-address: fd80:0:0:0::1
    next-hop-interface: eth1
    table-id: 253
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: true
    enabled: true
    dhcp: true
  ipv6:
    auto-dns: true
    enabled: true
    dhcp: true
- name: eth1
  type: ethernet
  state: down
`, dnsTestNicIP, dnsTestNic, dnsTestNic))
}

func dhcpAwareDNSAbsent(dnsTestNic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    server: []
    search: []
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    auto-dns: true
    enabled: true
    dhcp: true
  ipv6:
    auto-dns: true
    enabled: true
    dhcp: true
`, dnsTestNic))
}

var _ = Describe("Dns configuration", func() {
	Context("when desiredState is configured", func() {
		var (
			searchDomain1 = "fufu.ostest.test.metalkube.org"
			searchDomain2 = "sometest.com"
			server1V4     = "8.8.9.9"
			server1V6     = "2001:db8::1:2"
		)
		extractDNSServerAddress := func(dnsServer string) string {
			return strings.Split(dnsServer, "%")[0]
		}

		Context("with DHCP aware interface", func() {
			Context("with V4 upstream servers", func() {
				BeforeEach(func() {
					// read primary DNS server from one of the nodes
					serverList := dnsResolverForNode(nodes[0], "dns-resolver.running.server")
					updateDesiredStateAndWait(
						dhcpAwareDNSConfig(
							searchDomain1,
							searchDomain2,
							extractDNSServerAddress(serverList[0]),
							server1V4,
							dnsTestNic,
						),
					)
				})
				AfterEach(func() {
					updateDesiredStateAndWait(dhcpAwareDNSAbsent(dnsTestNic))
					for _, node := range nodes {
						dnsResolverServerForNodeEventually(node).ShouldNot(ContainElement(server1V4))
						dnsResolverSearchForNodeEventually(node).ShouldNot(ContainElement(searchDomain1))
						dnsResolverSearchForNodeEventually(node).ShouldNot(ContainElement(searchDomain2))
					}
					resetDesiredStateForNodes()
				})
				It("should have the static V4 address", func() {
					for _, node := range nodes {
						dnsResolverServerForNodeEventually(node).Should(ContainElement(server1V4))
						dnsResolverSearchForNodeEventually(node).Should(ContainElement(searchDomain1))
						dnsResolverSearchForNodeEventually(node).Should(ContainElement(searchDomain2))
					}
				})
			})
			Context("with V6 upstream servers", func() {
				BeforeEach(func() {
					// read primary DNS server from one of the nodes
					serverList := dnsResolverForNode(nodes[0], "dns-resolver.running.server")
					updateDesiredStateAndWait(
						dhcpAwareDNSConfig(
							searchDomain1,
							searchDomain2,
							extractDNSServerAddress(serverList[0]),
							server1V6,
							dnsTestNic,
						),
					)
				})
				AfterEach(func() {
					updateDesiredStateAndWait(dhcpAwareDNSAbsent(dnsTestNic))
					for _, node := range nodes {
						dnsResolverServerForNodeEventually(node).ShouldNot(ContainElement(server1V6))
						dnsResolverSearchForNodeEventually(node).ShouldNot(ContainElement(searchDomain1))
						dnsResolverSearchForNodeEventually(node).ShouldNot(ContainElement(searchDomain2))
					}
					resetDesiredStateForNodes()
				})
				It("should have the static V6 address", func() {
					for _, node := range nodes {
						dnsResolverServerForNodeEventually(node).Should(ContainElement(server1V6))
						dnsResolverSearchForNodeEventually(node).Should(ContainElement(searchDomain1))
						dnsResolverSearchForNodeEventually(node).Should(ContainElement(searchDomain2))
					}
				})
			})
		})
		XContext("with DHCP unaware interface, Skip Reason: https://bugzilla.redhat.com/show_bug.cgi?id=2054726", func() {
			var (
				designatedNode   string
				designatedNodeIP string
			)
			BeforeEach(func() {
				designatedNode = nodes[0]
				designatedNodeIP = ipv4Address(designatedNode, dnsTestNic)
			})
			Context("with V4 upstream servers", func() {
				BeforeEach(func() {
					// read primary DNS server from one of the nodes
					serverList := dnsResolverForNode(designatedNode, "dns-resolver.running.server")
					updateDesiredStateAtNodeAndWait(
						designatedNode,
						staticIPAndGwConfig(
							searchDomain1,
							searchDomain2,
							extractDNSServerAddress(serverList[0]),
							server1V4,
							dnsTestNic,
							designatedNodeIP,
						),
					)
				})
				AfterEach(func() {
					updateDesiredStateAtNodeAndWait(designatedNode, staticIPAbsentAndRoutesAbsent(dnsTestNic, designatedNodeIP))
					dnsResolverServerForNodeEventually(designatedNode).ShouldNot(ContainElement(server1V4))
					dnsResolverSearchForNodeEventually(designatedNode).ShouldNot(ContainElement(searchDomain1))
					dnsResolverSearchForNodeEventually(designatedNode).ShouldNot(ContainElement(searchDomain2))
					resetDesiredStateForNodes()
				})
				It("should have the static V4 address", func() {
					dnsResolverServerForNodeEventually(designatedNode).Should(ContainElement(server1V4))
					dnsResolverSearchForNodeEventually(designatedNode).Should(ContainElement(searchDomain1))
					dnsResolverSearchForNodeEventually(designatedNode).Should(ContainElement(searchDomain2))
				})
			})
			Context("with V6 upstream servers", func() {
				BeforeEach(func() {
					// read primary DNS server from one of the nodes
					serverList := dnsResolverForNode(designatedNode, "dns-resolver.running.server")
					updateDesiredStateAtNodeAndWait(
						designatedNode,
						staticIPAndGwConfig(
							searchDomain1,
							searchDomain2,
							extractDNSServerAddress(serverList[0]),
							server1V6,
							dnsTestNic,
							designatedNodeIP,
						),
					)
				})
				AfterEach(func() {
					updateDesiredStateAtNodeAndWait(designatedNode, staticIPAbsentAndRoutesAbsent(dnsTestNic, designatedNodeIP))
					dnsResolverServerForNodeEventually(designatedNode).ShouldNot(ContainElement(server1V6))
					dnsResolverSearchForNodeEventually(designatedNode).ShouldNot(ContainElement(searchDomain1))
					dnsResolverSearchForNodeEventually(designatedNode).ShouldNot(ContainElement(searchDomain2))
					resetDesiredStateForNodes()
				})
				It("should have the static V6 address", func() {
					dnsResolverServerForNodeEventually(designatedNode).Should(ContainElement(server1V6))
					dnsResolverSearchForNodeEventually(designatedNode).Should(ContainElement(searchDomain1))
					dnsResolverSearchForNodeEventually(designatedNode).Should(ContainElement(searchDomain2))
				})
			})
		})
	})
})
