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

package doc

import (
	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

type ExampleSpec struct {
	Name         string
	FileName     string
	PolicyName   string
	IfaceNames   []string
	CleanupState *nmstate.State
}

//nolint: funlen
func ExampleSpecs() []ExampleSpec {
	cleanDNSDesiredState := nmstate.NewState(`dns-resolver:
  config:
    search: []
    server: []
interfaces:
- name: eth1
  state: absent
`)

	cleanLLDPDesiredState := nmstate.NewState(`interfaces:
- name: eth0
  type: ethernet
  lldp:
    enabled: false
`)
	return []ExampleSpec{
		{
			Name:       "Ethernet",
			FileName:   "ethernet.yaml",
			PolicyName: "ethernet",
			IfaceNames: []string{"eth1"},
		},
		{
			Name:       "Linux bridge",
			FileName:   "linux-bridge.yaml",
			PolicyName: "linux-bridge",
			IfaceNames: []string{"br1"},
		},
		{
			Name:       "Linux bridge with custom vlan",
			FileName:   "linux-bridge-vlan.yaml",
			PolicyName: "linux-bridge-vlan",
			IfaceNames: []string{"br1"},
		},
		{
			Name:       "Detach bridge port and restore its configuration",
			FileName:   "detach-bridge-port-and-restore-eth.yaml",
			PolicyName: "detach-bridge-port-and-restore-eth",
			IfaceNames: []string{"br1"},
		},
		{
			Name:       "OVS bridge",
			FileName:   "ovs-bridge.yaml",
			PolicyName: "ovs-bridge",
			IfaceNames: []string{"br1"},
		},
		{
			Name:       "OVS bridge with interface",
			FileName:   "ovs-bridge-iface.yaml",
			PolicyName: "ovs-bridge-iface",
			IfaceNames: []string{"br1", "ovs0"},
		},
		{
			Name:       "Linux bonding",
			FileName:   "bond.yaml",
			PolicyName: "bond",
			IfaceNames: []string{"bond0"},
		},
		{
			Name:       "Linux bonding and VLAN",
			FileName:   "bond-vlan.yaml",
			PolicyName: "bond-vlan",
			IfaceNames: []string{"bond0.102", "bond0"},
		},
		{
			Name:       "VLAN",
			FileName:   "vlan.yaml",
			PolicyName: "vlan",
			IfaceNames: []string{"eth1.102", "eth1"},
		},
		{
			Name:       "DHCP",
			FileName:   "dhcp.yaml",
			PolicyName: "dhcp",
			IfaceNames: []string{"eth1"},
		},
		{
			Name:       "Static IP",
			FileName:   "static-ip.yaml",
			PolicyName: "static-ip",
			IfaceNames: []string{"eth1"},
		},
		{
			Name:       "Route",
			FileName:   "route.yaml",
			PolicyName: "route",
			IfaceNames: []string{"eth1"},
		},
		{
			Name:         "DNS",
			FileName:     "dns.yaml",
			PolicyName:   "dns",
			IfaceNames:   []string{},
			CleanupState: &cleanDNSDesiredState,
		},
		{
			Name:         "LLDP",
			FileName:     "lldp.yaml",
			PolicyName:   "enable-lldp-ethernets-up",
			IfaceNames:   []string{},
			CleanupState: &cleanLLDPDesiredState,
		},
		{
			Name:       "Worker selector",
			FileName:   "worker-selector.yaml",
			PolicyName: "worker-selector",
			IfaceNames: []string{"eth1"},
		},
	}
}
