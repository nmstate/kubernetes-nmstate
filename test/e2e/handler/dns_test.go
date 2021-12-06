package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func dnsConfig(searchDomain1, searchDomain2, server1, server2, ipFamily, ipFamily1, dnsTestNic string) nmstate.State {
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
  %s:
    auto-dns: false
    enabled: true
    dhcp: true
  %s:
    enabled: true
    dhcp: true
    auto-dns: false

`, searchDomain1, searchDomain2, server1, server2, dnsTestNic, ipFamily, ipFamily1))
}

func dnsAbsent(dnsTestNic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`dns-resolver:
  config:
    state: absent
interfaces:
- name: %s
  type: ethernet
  state: up
  ipv4:
    dhcp: true
    enabled: true
    auto-dns: true
  ipv6:
    dhcp: true
    enabled: true
    auto-dns: true
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

		Context("with V4 upstream servers", func() {
			BeforeEach(func() {
				// read primary DNS server from one of the nodes
				serverList := dnsResolverForNode(nodes[0], "dns-resolver.running.server")
				updateDesiredStateAndWait(dnsConfig(searchDomain1, searchDomain2, serverList[0], server1V4, "ipv4", "ipv6", dnsTestNic))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(dnsAbsent(dnsTestNic))
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
				updateDesiredStateAndWait(dnsConfig(searchDomain1, searchDomain2, serverList[0], server1V6, "ipv6", "ipv4", dnsTestNic))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(dnsAbsent(dnsTestNic))
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
})
