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

package probe

import (
	"net"
	"testing"
)

// nolint: funlen
func TestDefaultGw4Parsing(t *testing.T) {
	tests := []struct {
		desc          string
		status        string
		expectedGw    string
		expectedIface string
		shouldErr     bool
	}{
		{
			desc: "one single IPv4 gateway",
			status: `routes:
  running:
  - destination: 172.30.0.0/16
    next-hop-interface: eth0
    next-hop-address: 169.254.169.4
    table-id: 254
  - destination: 0.0.0.0/0
    next-hop-interface: eth1
    next-hop-address: 10.46.55.254
    metric: 48
    table-id: 254
`,
			expectedGw:    "10.46.55.254",
			expectedIface: "eth1",
		}, {
			desc: "two IPv4 gateways, one on custom routing table",
			status: `routes:
  running:
  - destination: 172.30.0.0/16
    next-hop-interface: eth0
    next-hop-address: 169.254.169.4
    table-id: 254
  - destination: 0.0.0.0/0
    next-hop-interface: eth0
    next-hop-address: 169.254.169.4
    table-id: 56
  - destination: 0.0.0.0/0
    next-hop-interface: eth1
    next-hop-address: 10.46.55.254
    metric: 48
    table-id: 254
`,
			expectedGw:    "10.46.55.254",
			expectedIface: "eth1",
		}, {
			desc: "no IPv4 default gateway",
			status: `routes:
  running:
  - destination: 172.30.0.0/16
    next-hop-interface: eth0
    next-hop-address: 169.254.169.4
    table-id: 254
`,
			shouldErr: true,
		}, {
			desc: "only IPv6 gateway present",
			status: `routes:
  running:
  - destination: ::/0
    next-hop-interface: eth0
    next-hop-address: fe80::dead:beef:fe51:782d
    table-id: 254
`,
			shouldErr: true,
		}, {
			desc: "dual-stack picks IPv4 gateway",
			status: `routes:
  running:
  - destination: 0.0.0.0/0
    next-hop-interface: eth0
    next-hop-address: 10.46.55.254
    table-id: 254
  - destination: ::/0
    next-hop-interface: eth0
    next-hop-address: fe80::dead:beef:fe51:782d
    table-id: 254
`,
			expectedGw:    "10.46.55.254",
			expectedIface: "eth0",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gJSON, err := yamlToGJson(test.status)
			if err != nil {
				t.Fatalf("failed to parse test status, %v", err)
			}
			gw, err := defaultGw4(gJSON)
			if err != nil && !test.shouldErr {
				t.Fatalf("unexpected error %v", err)
			}
			if test.shouldErr && err == nil {
				t.Fatalf("expecting error, did not fail")
			}

			expectedRoute := Route{
				nextHop: net.ParseIP(test.expectedGw),
				iface:   test.expectedIface,
			}

			if !expectedRoute.nextHop.Equal(gw.nextHop) || expectedRoute.iface != gw.iface {
				t.Fatalf("expecting %+v, got %+v", expectedRoute, gw)
			}
		})
	}
}

// nolint: funlen
func TestDefaultGw6Parsing(t *testing.T) {
	tests := []struct {
		desc          string
		status        string
		expectedGw    string
		expectedIface string
		shouldErr     bool
	}{
		{
			desc: "one single IPv6 gateway",
			status: `routes:
  running:
  - destination: ::/0
    next-hop-interface: eth0
    next-hop-address: fe80::dead:beef:fe51:782d
    table-id: 254
`,
			expectedGw:    "fe80::dead:beef:fe51:782d",
			expectedIface: "eth0",
		}, {
			desc: "two IPv6 gateways, one on custom routing table",
			status: `routes:
  running:
  - destination: ::/0
    next-hop-interface: eth0
    next-hop-address: fe80::dead:beef:fe51:782d
    table-id: 254
  - destination: ::/0
    next-hop-interface: eth1
    next-hop-address: fe80::baad:cafe:fe51:782d
    table-id: 56
`,
			expectedGw:    "fe80::dead:beef:fe51:782d",
			expectedIface: "eth0",
		}, {
			desc: "no IPv6 default gateway",
			status: `routes:
  running:
  - destination: 172.30.0.0/16
    next-hop-interface: eth0
    next-hop-address: 169.254.169.4
    table-id: 254
`,
			shouldErr: true,
		}, {
			desc: "only IPv4 gateway present",
			status: `routes:
  running:
  - destination: 0.0.0.0/0
    next-hop-interface: eth1
    next-hop-address: 10.46.55.254
    table-id: 254
`,
			shouldErr: true,
		}, {
			desc: "dual-stack picks IPv6 gateway",
			status: `routes:
  running:
  - destination: 0.0.0.0/0
    next-hop-interface: eth0
    next-hop-address: 10.46.55.254
    table-id: 254
  - destination: ::/0
    next-hop-interface: eth0
    next-hop-address: fe80::dead:beef:fe51:782d
    table-id: 254
`,
			expectedGw:    "fe80::dead:beef:fe51:782d",
			expectedIface: "eth0",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gJSON, err := yamlToGJson(test.status)
			if err != nil {
				t.Fatalf("failed to parse test status, %v", err)
			}
			gw, err := defaultGw6(gJSON)
			if err != nil && !test.shouldErr {
				t.Fatalf("unexpected error %v", err)
			}
			if test.shouldErr && err == nil {
				t.Fatalf("expecting error, did not fail")
			}

			expectedRoute := Route{
				nextHop: net.ParseIP(test.expectedGw),
				iface:   test.expectedIface,
			}

			if !expectedRoute.nextHop.Equal(gw.nextHop) || expectedRoute.iface != gw.iface {
				t.Fatalf("expecting %+v, got %+v", expectedRoute, gw)
			}
		})
	}
}
