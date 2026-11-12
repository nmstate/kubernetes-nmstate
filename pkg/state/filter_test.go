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

package state

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("FilterOut", func() {
	var (
		state, filteredState nmstate.State
	)

	Context("when there is a linux bridge with gc-timer and hello-timer", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: eth1
  state: up
  type: ethernet
- name: br1
  bridge:
    options:
      gc-timer: 13715
      group-addr: 01:80:C2:00:00:00
      group-forward-mask: 0
      hash-max: 512
      hello-timer: 0
      stp:
        enabled: false
    port: []
  ipv4:
    address:
    - ip: 172.17.0.1
      prefix-length: 16
    dhcp: false
    enabled: true
  ipv6:
    address:
    - ip: 2001:db9:1::1
      prefix-length: 64
    - ip: fe80::1
      prefix-length: 64
    autoconf: false
    dhcp: false
    enabled: true
  lldp:
    enabled: false
  mac-address: 02:42:BB:10:B8:9F
  mtu: 1500
  state: up
  type: linux-bridge
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
			filteredState = nmstate.NewState(`
interfaces:
- name: eth1
  state: up
  type: ethernet
- name: br1
  bridge:
    options:
      group-addr: 01:80:C2:00:00:00
      group-forward-mask: 0
      hash-max: 512
      stp:
        enabled: false
    port: []
  ipv4:
    address:
    - ip: 172.17.0.1
      prefix-length: 16
    dhcp: false
    enabled: true
  ipv6:
    address:
    - ip: 2001:db9:1::1
      prefix-length: 64
    - ip: fe80::1
      prefix-length: 64
    autoconf: false
    dhcp: false
    enabled: true
  lldp:
    enabled: false
  mac-address: 02:42:BB:10:B8:9F
  mtu: 1500
  state: up
  type: linux-bridge
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
		})
		It("should remove dynamic attributes from linux-bridge interface", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when there is managed veth interface", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: vethab6030bd
  state: down
  type: veth
  veth:
    peer: eth2
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
    table-id: 254
`)
			filteredState = nmstate.NewState(`interfaces:
- name: vethab6030bd
  state: down
  type: veth
  veth:
    peer: eth2
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
    table-id: 254
`)
		})

		It("should keep managed veth interface", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when there is unmanaged veth interface", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: vethab6030bd
  state: ignore
  type: veth
  veth:
    peer: eth2
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
    table-id: 254
`)
			filteredState = nmstate.NewState(`interfaces: []
routes:
  config: []
  running: []
`)
		})

		It("should filter unmanaged veth interface", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when there are multiple managed and unmanaged interfaces", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: veth101
  state: down
  type: veth
  veth:
    peer: eth2
- name: veth102
  state: ignore
  type: veth
  veth:
    peer: eth2
- name: vethjyuftrgv
  state: down
  type: veth
  veth:
    peer: eth2
- name: vethvasziovs
  state: ignore
  type: veth
  veth:
    peer: eth2
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: veth101
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: veth102
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethjyuftrgv
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethvasziovs
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
			filteredState = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: veth101
  state: down
  type: veth
  veth:
    peer: eth2
- name: vethjyuftrgv
  state: down
  type: veth
  veth:
    peer: eth2
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: veth101
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethjyuftrgv
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
		})
		It("should filter out all unmanaged veth interfaces", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("With DNS Resolver populated", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
  - name: eth1
    state: up
    type: ethernet
dns-resolver:
  config:
    search:
    - example.com
    - example.org
    server:
    - 2001:4860:4860::8888
    - 8.8.8.8
  running:
    search:
    - example.running.com
    - example.running.org
    server:
    - 8.8.4.4`)
		})

		It("Should keep the DNS Resolver intact", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(state))
		})
	})

	Context("when there are interfaces with preferred-life-time and valid-life-time in addresses", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: eth1
  state: up
  type: ethernet
  ipv4:
    address:
    - ip: 192.168.1.1
      prefix-length: 24
      preferred-life-time: 3600
      valid-life-time: 7200
    dhcp: false
    enabled: true
  ipv6:
    address:
    - ip: 2001:db8::1
      prefix-length: 64
      preferred-life-time: 1800
      valid-life-time: 3600
    - ip: fe80::1
      prefix-length: 64
      preferred-life-time: forever
      valid-life-time: forever
    autoconf: false
    dhcp: false
    enabled: true
routes:
  config: []
  running: []
`)
			filteredState = nmstate.NewState(`
interfaces:
- name: eth1
  state: up
  type: ethernet
  ipv4:
    address:
    - ip: 192.168.1.1
      prefix-length: 24
    dhcp: false
    enabled: true
  ipv6:
    address:
    - ip: 2001:db8::1
      prefix-length: 64
    - ip: fe80::1
      prefix-length: 64
    autoconf: false
    dhcp: false
    enabled: true
routes:
  config: []
  running: []
`)
		})
		It("should remove preferred-life-time and valid-life-time from address entries", func() {
			returnedState, _, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the serialized state fits within the size budget", func() {
		BeforeEach(func() {
			state = stateWithVlans(3)
		})
		It("should keep VLAN interfaces intact and report no compaction", func() {
			returnedState, tier, err := filterOut(state, DefaultMaxStateSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(tier).To(Equal(0))
			Expect(returnedState).To(MatchYAML(state))
		})
	})

	Context("when the serialized state exceeds the size budget", func() {
		// Enough for the ethernet interface plus three VLANs stripped of
		// their verbose fields, but not for three full VLANs.
		const maxSizeBytes = 4000

		BeforeEach(func() {
			state = stateWithVlans(3)
			filteredState = stateWithVlansRendered(3, compactVlanInterface)
		})
		It("should strip verbose fields from VLAN interfaces and keep other interfaces intact", func() {
			returnedState, tier, err := filterOut(state, maxSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(tier).To(Equal(1))
			Expect(returnedState).To(MatchYAML(filteredState))
			Expect(serializedSizeOf(returnedState)).To(BeNumerically("<=", maxSizeBytes))
		})
	})

	Context("when stripping verbose VLAN fields is not enough to fit the size budget", func() {
		// Enough for the ethernet interface plus three VLANs reduced to
		// their identity and addresses, but not for three VLANs that only
		// had their verbose fields stripped.
		const maxSizeBytes = 2400

		BeforeEach(func() {
			state = stateWithVlans(3)
			filteredState = stateWithVlansRendered(3, minimalVlanInterface)
		})
		It("should reduce VLAN interfaces to their identity and addresses", func() {
			returnedState, tier, err := filterOut(state, maxSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(tier).To(Equal(2))
			Expect(returnedState).To(MatchYAML(filteredState))
			Expect(serializedSizeOf(returnedState)).To(BeNumerically("<=", maxSizeBytes))
		})
	})

	Context("when no compaction is enough to fit the size budget", func() {
		const maxSizeBytes = 1

		BeforeEach(func() {
			state = stateWithVlans(3)
			filteredState = stateWithVlansRendered(3, minimalVlanInterface)
		})
		It("should return the most compact state without dropping interfaces", func() {
			returnedState, tier, err := filterOut(state, maxSizeBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(tier).To(Equal(2))
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the size budget is not positive", func() {
		BeforeEach(func() {
			state = stateWithVlans(3)
		})
		It("should fall back to the default size budget and keep a small state intact", func() {
			returnedState, tier, err := filterOut(state, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(tier).To(Equal(0))
			Expect(returnedState).To(MatchYAML(state))
		})
	})
})

// serializedSizeOf returns the size of the state as stored in the
// NodeNetworkState, which is JSON (see shared.State.MarshalJSON).
func serializedSizeOf(state nmstate.State) int {
	jsonState, err := yaml.YAMLToJSON(state.Raw)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return len(jsonState)
}

// realisticEthernetInterface mirrors `nmstatectl show` output for a
// NetworkManager-managed ethernet interface on RHCOS (nmstate 2.2.60).
func realisticEthernetInterface(name string) string {
	return fmt.Sprintf(`- name: %s
  type: ethernet
  profile-name: %s
  driver: virtio_net
  state: up
  identifier: name
  mac-address: 00:6B:3D:CE:C4:E6
  permanent-mac-address: 00:6B:3D:CE:C4:E6
  mtu: 1500
  min-mtu: 68
  max-mtu: 65535
  ipv4:
    enabled: false
    forwarding: true
  ipv6:
    enabled: false
  accept-all-mac-addresses: false
  lldp:
    enabled: false
  ethtool:
    feature:
      rx-gro: true
      rx-gro-hw: true
      rx-gro-list: false
      rx-udp-gro-forwarding: true
      tx-checksum-ip-generic: true
      tx-generic-segmentation: true
      tx-nocache-copy: false
      tx-tcp-ecn-segmentation: true
      tx-tcp-mangleid-segmentation: false
      tx-tcp-segmentation: true
      tx-tcp6-segmentation: true
    coalesce:
      adaptive-rx: false
      rx-frames: 1
      rx-usecs: 0
      tx-frames: 1
      tx-usecs: 0
    ring:
      rx: 256
      rx-max: 256
      tx: 256
      tx-max: 256
  alt-names:
  - name: enx006b3dcec4e6
  ethernet:
    auto-negotiation: false
`, name, name)
}

// realisticVlanInterface mirrors `nmstatectl show` output for a
// NetworkManager-managed VLAN with a static IPv4 address and IPv6
// autoconf on RHCOS (nmstate 2.2.60).
func realisticVlanInterface(baseIface string, id int) string {
	name := fmt.Sprintf("%s.%d", baseIface, id)
	return fmt.Sprintf(`- name: %[1]s
  profile-name: %[1]s
  type: vlan
  state: up
  identifier: name
  mac-address: 00:6B:3D:CE:C4:E6
  mtu: 1500
  max-mtu: 65535
  wait-ip: any
  ipv4:
    enabled: true
    dhcp: false
    address:
    - ip: 192.0.%[3]d.%[4]d
      prefix-length: 24
    auto-route-metric: -1
    forwarding: true
    prefix-route-metric: -1
  ipv6:
    enabled: true
    dhcp: true
    autoconf: true
    address:
    - ip: fe80::4f9a:1ed2:3812:8e49
      prefix-length: 64
    auto-dns: true
    auto-gateway: true
    auto-routes: true
    auto-route-table-id: 0
    auto-route-metric: -1
    addr-gen-mode: stable-privacy
    dhcp-send-hostname: true
  mptcp:
    address-flags: []
  accept-all-mac-addresses: false
  lldp:
    enabled: false
  ethtool:
    feature:
      highdma: true
      rx-gro: true
      rx-gro-list: false
      rx-udp-gro-forwarding: false
      tx-checksum-fcoe-crc: false
      tx-checksum-ip-generic: true
      tx-checksum-sctp: false
      tx-fcoe-segmentation: false
      tx-generic-segmentation: true
      tx-gre-csum-segmentation: false
      tx-gre-segmentation: false
      tx-gso-list: true
      tx-ipxip4-segmentation: false
      tx-ipxip6-segmentation: false
      tx-nocache-copy: false
      tx-scatter-gather-fraglist: false
      tx-sctp-segmentation: true
      tx-tcp-ecn-segmentation: true
      tx-tcp-mangleid-segmentation: true
      tx-tcp-segmentation: true
      tx-tcp6-segmentation: true
      tx-udp-segmentation: true
      tx-udp_tnl-csum-segmentation: false
      tx-udp_tnl-segmentation: false
  vlan:
    base-iface: %[2]s
    id: %[5]d
    protocol: 802.1q
    registration-protocol: none
    reorder-headers: true
    loose-binding: false
`, name, baseIface, id/256, id%256, id)
}

// compactVlanInterface is realisticVlanInterface reduced to the fields kept
// when the state exceeds the size budget.
func compactVlanInterface(baseIface string, id int) string {
	return fmt.Sprintf(`- name: %[1]s.%[4]d
  type: vlan
  state: up
  ipv4:
    enabled: true
    dhcp: false
    address:
    - ip: 192.0.%[2]d.%[3]d
      prefix-length: 24
    auto-route-metric: -1
    forwarding: true
    prefix-route-metric: -1
  ipv6:
    enabled: true
    dhcp: true
    autoconf: true
    address:
    - ip: fe80::4f9a:1ed2:3812:8e49
      prefix-length: 64
    auto-dns: true
    auto-gateway: true
    auto-routes: true
    auto-route-table-id: 0
    auto-route-metric: -1
    addr-gen-mode: stable-privacy
    dhcp-send-hostname: true
  vlan:
    base-iface: %[1]s
    id: %[4]d
    protocol: 802.1q
    registration-protocol: none
    reorder-headers: true
    loose-binding: false
`, baseIface, id/256, id%256, id)
}

// minimalVlanInterface is realisticVlanInterface reduced to the fields kept
// when stripping verbose fields is not enough to fit the size budget.
func minimalVlanInterface(baseIface string, id int) string {
	return fmt.Sprintf(`- name: %[1]s.%[4]d
  type: vlan
  state: up
  ipv4:
    enabled: true
    dhcp: false
    address:
    - ip: 192.0.%[2]d.%[3]d
      prefix-length: 24
  ipv6:
    enabled: true
    dhcp: true
    autoconf: true
    address:
    - ip: fe80::4f9a:1ed2:3812:8e49
      prefix-length: 64
  vlan:
    base-iface: %[1]s
    id: %[4]d
`, baseIface, id/256, id%256, id)
}

// stateWithVlans builds a state with one ethernet interface and the given
// number of VLAN interfaces on top of it.
func stateWithVlans(count int) nmstate.State {
	return stateWithVlansRendered(count, realisticVlanInterface)
}

func stateWithVlansRendered(count int, renderVlan func(baseIface string, id int) string) nmstate.State {
	var sb strings.Builder
	sb.WriteString("interfaces:\n")
	sb.WriteString(realisticEthernetInterface("eth0"))
	for id := 1000; id < 1000+count; id++ {
		sb.WriteString(renderVlan("eth0", id))
	}
	sb.WriteString("routes:\n  config: []\n  running: []\n")
	return nmstate.NewState(sb.String())
}

var _ = Describe("CountRoutes", func() {
	Context("when there are no routes", func() {
		It("should return empty map", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(BeEmpty())
		})
	})

	Context("when there are only IPv4 dynamic routes", func() {
		It("should count them correctly", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
  - destination: 192.168.66.0/24
    metric: 100
    next-hop-address: ""
    next-hop-interface: eth1
    table-id: 254
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(HaveLen(1))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "dynamic"}]).To(Equal(2))
		})
	})

	Context("when there are only IPv6 dynamic routes", func() {
		It("should count them correctly", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: eth1
    table-id: 254
  - destination: ::/0
    metric: 100
    next-hop-address: fe80::1
    next-hop-interface: eth1
    table-id: 254
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(HaveLen(1))
			Expect(counts[RouteKey{IPStack: "ipv6", Type: "dynamic"}]).To(Equal(2))
		})
	})

	Context("when there are mixed IPv4 and IPv6 routes", func() {
		It("should count them by IP stack", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: eth1
    table-id: 254
  - destination: ::/0
    metric: 100
    next-hop-address: fe80::1
    next-hop-interface: eth1
    table-id: 254
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(HaveLen(2))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "dynamic"}]).To(Equal(1))
			Expect(counts[RouteKey{IPStack: "ipv6", Type: "dynamic"}]).To(Equal(2))
		})
	})

	Context("when there are static and dynamic routes", func() {
		It("should distinguish static routes correctly", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
routes:
  config:
  - destination: 10.0.0.0/8
    metric: 100
    next-hop-address: 192.168.66.1
    next-hop-interface: eth1
    table-id: 254
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
  - destination: 10.0.0.0/8
    metric: 100
    next-hop-address: 192.168.66.1
    next-hop-interface: eth1
    table-id: 254
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(HaveLen(2))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "static"}]).To(Equal(1))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "dynamic"}]).To(Equal(1))
		})
	})

	Context("when there are static and dynamic routes for both IPv4 and IPv6", func() {
		It("should count all combinations correctly", func() {
			state := nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
routes:
  config:
  - destination: 10.0.0.0/8
    metric: 100
    next-hop-address: 192.168.66.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 2001:db8::/32
    metric: 100
    next-hop-address: fe80::1
    next-hop-interface: eth1
    table-id: 254
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
  - destination: 10.0.0.0/8
    metric: 100
    next-hop-address: 192.168.66.1
    next-hop-interface: eth1
    table-id: 254
  - destination: ::/0
    metric: 100
    next-hop-address: fe80::1
    next-hop-interface: eth1
    table-id: 254
  - destination: 2001:db8::/32
    metric: 100
    next-hop-address: fe80::1
    next-hop-interface: eth1
    table-id: 254
`)
			counts, err := CountRoutes(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(counts).To(HaveLen(4))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "static"}]).To(Equal(1))
			Expect(counts[RouteKey{IPStack: "ipv4", Type: "dynamic"}]).To(Equal(1))
			Expect(counts[RouteKey{IPStack: "ipv6", Type: "static"}]).To(Equal(1))
			Expect(counts[RouteKey{IPStack: "ipv6", Type: "dynamic"}]).To(Equal(1))
		})
	})
})
