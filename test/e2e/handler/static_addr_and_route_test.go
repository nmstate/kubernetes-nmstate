package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func ipV4AddrAndRoute(firstSecondaryNic, ipAddress, destIPAddress, prefixLen, nextHopIPAddress string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      address:
      - ip: %s
        prefix-length: %s
      dhcp: false
      enabled: true
routes:
    config:
    - destination: %s
      metric: 150
      next-hop-address: %s
      next-hop-interface: %s
      table-id: 254
`, firstSecondaryNic, ipAddress, prefixLen, destIPAddress, nextHopIPAddress, firstSecondaryNic))
}

func ipV4AddrAndRouteAbsent(firstSecondaryNic, ipAddress, prefixLen string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      address:
      - ip: %s
        prefix-length: %s
      state: absent
      enabled: false
routes:
    config:
    - next-hop-interface: %s
      state: absent
`, firstSecondaryNic, ipAddress, prefixLen, firstSecondaryNic))
}

func ipV6Addr(firstSecondaryNic, ipAddressV6, prefixLenV6 string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv6:
      address:
      - ip: %s
        prefix-length: %s
      dhcp: false
      enabled: true
`, firstSecondaryNic, ipAddressV6, prefixLenV6))
}

func ipV6AddrAbsent(firstSecondaryNic, ipAddressV6, prefixLenV6 string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv6:
      address:
      - ip: %s
        prefix-length: %s
      state: absent
      enabled: false
`, firstSecondaryNic, ipAddressV6, prefixLenV6))
}

func ipV6AddrAndRoute(firstSecondaryNic, ipAddressV6, destIPAddressV6, prefixLenV6, nextHopIPAddressV6 string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv6:
      address:
      - ip: %s
        prefix-length: %s
      dhcp: false
      enabled: true
routes:
    config:
    - destination: %s
      metric: 150
      next-hop-address: %s
      next-hop-interface: %s
      table-id: 254
`, firstSecondaryNic, ipAddressV6, prefixLenV6, destIPAddressV6, nextHopIPAddressV6, firstSecondaryNic))
}

func ipV6AddrAndRouteAbsent(firstSecondaryNic, ipAddressV6, prefixLenV6 string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv6:
      address:
      - ip: %s
        prefix-length: %s
      state: absent
      enabled: false
routes:
    config:
    - next-hop-interface: %s
      state: absent
`, firstSecondaryNic, ipAddressV6, prefixLenV6, firstSecondaryNic))
}

var _ = Describe("Static addresses and routes", func() {
	Context("when desiredState is configured", func() {
		var (
			ipAddress          = "192.0.2.251"
			destIPAddress      = "198.51.100.0/24"
			prefixLen          = "24"
			nextHopIPAddress   = "192.0.2.1"
			ipAddressV6        = "2001:db8::1:1"
			prefixLenV6        = "64"
			destIPAddressV6    = "2001:dc8::/64"
			nextHopIPAddressV6 = "2001:db8::1:2"
		)

		Context("with static V4 address", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(ifaceUpWithStaticIP(firstSecondaryNic, ipAddress, prefixLen))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(ifaceUpWithStaticIPAbsent(firstSecondaryNic, ipAddress, prefixLen))
				for _, node := range nodes {
					ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).ShouldNot(Equal(ipAddress))
				}
				resetDesiredStateForNodes()
			})
			It("should have the static V4 address", func() {
				for _, node := range nodes {
					ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddress))
				}
			})
		})

		Context("with static V4 address and route", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(ipV4AddrAndRoute(firstSecondaryNic, ipAddress, destIPAddress, prefixLen, nextHopIPAddress))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(ipV4AddrAndRouteAbsent(firstSecondaryNic, ipAddress, prefixLen))
				for _, node := range nodes {
					ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).ShouldNot(Equal(ipAddress))
					routeDestForNodeInterfaceEventually(node, destIPAddress).ShouldNot(Equal(firstSecondaryNic))
				}
				resetDesiredStateForNodes()
			})
			It("should have the static V4 address and route  at currentState", func() {
				for _, node := range nodes {
					ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddress))
					routeNextHopInterface(node, destIPAddress).Should(Equal(firstSecondaryNic))
				}
			})
		})

		Context("with static V6 address", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(ipV6Addr(firstSecondaryNic, ipAddressV6, prefixLenV6))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(ipV6AddrAbsent(firstSecondaryNic, ipAddressV6, prefixLenV6))
				for _, node := range nodes {
					ipV6AddressForNodeInterfaceEventually(node, firstSecondaryNic).ShouldNot(Equal(ipAddressV6))
				}
				resetDesiredStateForNodes()
			})
			It("should have the static V6 address", func() {
				for _, node := range nodes {
					ipV6AddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddressV6))
				}
			})
		})

		Context("with static V6 address and route", func() {
			BeforeEach(func() {
				updateDesiredStateAndWait(ipV6AddrAndRoute(firstSecondaryNic, ipAddressV6, destIPAddressV6, prefixLenV6, nextHopIPAddressV6))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(ipV6AddrAndRouteAbsent(firstSecondaryNic, ipAddressV6, prefixLenV6))
				for _, node := range nodes {
					ipV6AddressForNodeInterfaceEventually(node, firstSecondaryNic).ShouldNot(Equal(ipAddressV6))
					routeDestForNodeInterfaceEventually(node, destIPAddressV6).ShouldNot(Equal(firstSecondaryNic))
				}
				resetDesiredStateForNodes()
			})
			It("should have the static V6 address and route  at currentState", func() {
				for _, node := range nodes {
					ipV6AddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddressV6))
					routeNextHopInterface(node, destIPAddressV6).Should(Equal(firstSecondaryNic))
				}
			})
		})
	})
})
