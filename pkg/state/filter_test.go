package state

import (
	"github.com/gobwas/glob"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("FilterOut", func() {
	var (
		state, filteredState nmstate.State
		interfacesFilterGlob glob.Glob
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
      mac-ageing-time: 300
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
  name: br1
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
      mac-ageing-time: 300
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
  name: br1
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
			interfacesFilterGlob = glob.MustCompile("")
		})
		It("should remove them from linux-bridge", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})
	Context("when the filter is set to empty and there is a list of interfaces", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
			interfacesFilterGlob = glob.MustCompile("")
		})

		It("should keep all interfaces intact", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(state))
		})
	})

	Context("when the filter is matching one of the interfaces in the list", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
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
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
			interfacesFilterGlob = glob.MustCompile("veth*")
		})

		It("should filter out matching interface and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the filter is matching multiple interfaces in the list", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
- name: vethjyuftrgv
  state: down
  type: ethernet
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
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
			filteredState = nmstate.NewState(`interfaces:
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
`)
			interfacesFilterGlob = glob.MustCompile("veth*")
		})

		It("should filter out all matching interfaces and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the filter is matching multiple prefixes", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
- name: vnet2b730a2b@if3
  state: down
  type: ethernet
routes:
  config: []
  running:
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vethab6030bd
    table-id: 254
  - destination: fd10:244::8c40/128
    metric: 1024
    next-hop-address: ""
    next-hop-interface: vnet2b730a2b
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
routes:
  config: []
  running:
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`)
			interfacesFilterGlob = glob.MustCompile("{veth*,vnet*}")
		})

		It("it should filter out all interfaces matching any of these prefixes and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})
	Context("when there is a linux bridge without 'bridge' options because is down", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: br1
  type: linux-bridge
  state: down
`)

			filteredState = nmstate.NewState(`
interfaces:
- name: br1
  type: linux-bridge
  state: down
`)
			interfacesFilterGlob = glob.MustCompile("")
		})
		It("should keep the bridge as it is", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	Context("when the interfaces has numeric characters quoted", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: eth0
- name: '0'
- name: '1101010'
- name: '0.0'
- name: '1.0'
- name: '0xfe'
- name: '60.e+02'
`)
			filteredState = nmstate.NewState(`
interfaces:
- name: eth0
- name: '1101010'
- name: '1.0'
- name: '60.e+02'
`)
			interfacesFilterGlob = glob.MustCompile("0*")
		})

		It("should filter out interfaces correctly", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	// See https://github.com/yaml/pyyaml/issues/173 for why this scenario is checked.
	Context("when the interfaces names have numbers in scientific notation without dot", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces:
- name: eth0
- name: 10e+02
- name: 60e+02
`)
			filteredState = nmstate.NewState(`
interfaces:
- name: eth0
- name: "60e+02"
`)
			interfacesFilterGlob = glob.MustCompile("10e*")
		})

		It("does not filter out interfaces correctly and does not represent them correctly", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})
})
