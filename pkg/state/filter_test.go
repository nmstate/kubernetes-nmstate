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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"sigs.k8s.io/yaml"
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
			returnedState, err := filterOut(state)
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
			returnedState, err := filterOut(state)
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
			returnedState, err := filterOut(state)
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
			returnedState, err := filterOut(state)
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
			returnedState, err := filterOut(state)
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
			returnedState, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the number of interfaces exceeds the threshold", func() {
		It("should strip verbose fields from VLAN interfaces but keep essential ones", func() {
			// Build a state with more than interfaceCountThreshold interfaces
			// by generating VLAN interfaces
			yamlStr := "interfaces:\n"
			yamlStr += "- name: eth0\n  state: up\n  type: ethernet\n  mtu: 1500\n  mac-address: 00:11:22:33:44:55\n"
			for i := 0; i < interfaceCountThreshold+10; i++ {
				yamlStr += fmt.Sprintf(`- name: eth0.%d
  type: vlan
  state: up
  mtu: 1400
  mac-address: 02:00:00:%02x:%02x:00
  ipv4:
    enabled: true
    address:
    - ip: 192.168.%d.1
      prefix-length: 24
  vlan:
    id: %d
    base-iface: eth0
  lldp:
    enabled: false
  ethtool:
    feature:
      tx-checksum-ip-generic: true
`, i, i/256, i%256, i%256, i)
			}
			yamlStr += "routes:\n  config: []\n  running: []\n"

			state := nmstate.NewState(yamlStr)
			result, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())

			// Parse the result to verify
			var parsed rootState
			err = yaml.Unmarshal(result.Raw, &parsed)
			Expect(err).ToNot(HaveOccurred())

			// Ethernet interface should be untouched
			eth0 := parsed.Interfaces[0]
			Expect(eth0.Name).To(Equal("eth0"))
			Expect(eth0.Data).To(HaveKey("mtu"))
			Expect(eth0.Data).To(HaveKey("mac-address"))

			// VLAN interfaces should have verbose fields stripped
			vlan0 := parsed.Interfaces[1]
			Expect(vlan0.Name).To(Equal("eth0.0"))
			Expect(vlan0.Type).To(Equal("vlan"))
			Expect(vlan0.Data).To(HaveKey("state"))
			Expect(vlan0.Data).To(HaveKey("vlan"))
			Expect(vlan0.Data).To(HaveKey("ipv4"))
			// Verbose fields should be gone
			Expect(vlan0.Data).NotTo(HaveKey("mtu"))
			Expect(vlan0.Data).NotTo(HaveKey("mac-address"))
			Expect(vlan0.Data).NotTo(HaveKey("lldp"))
			Expect(vlan0.Data).NotTo(HaveKey("ethtool"))
		})

		It("should not strip fields when interface count is below threshold", func() {
			state := nmstate.NewState(`
interfaces:
- name: eth0
  state: up
  type: ethernet
  mtu: 1500
- name: eth0.100
  type: vlan
  state: up
  mtu: 1400
  mac-address: 02:00:00:00:64:00
  ipv4:
    enabled: true
  vlan:
    id: 100
    base-iface: eth0
  lldp:
    enabled: false
routes:
  config: []
  running: []
`)
			result, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())

			var parsed rootState
			err = yaml.Unmarshal(result.Raw, &parsed)
			Expect(err).ToNot(HaveOccurred())

			// VLAN should retain all fields when below threshold
			vlan := parsed.Interfaces[1]
			Expect(vlan.Data).To(HaveKey("mtu"))
			Expect(vlan.Data).To(HaveKey("mac-address"))
			Expect(vlan.Data).To(HaveKey("lldp"))
		})
	})
})

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
