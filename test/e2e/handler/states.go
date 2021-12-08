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
	"text/template"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func ethernetNicsState(states map[string]string) nmstate.State {
	tmp, err := template.New("ethernetNicsUp").Parse(`interfaces:
{{ range $nic, $state := . }}
  - name: {{ $nic }}
    type: ethernet
    state: {{ $state }}
{{ end }}
`)
	Expect(err).ToNot(HaveOccurred())

	stringBuilder := strings.Builder{}
	err = tmp.Execute(&stringBuilder, states)
	Expect(err).ToNot(HaveOccurred())

	return nmstate.NewState(stringBuilder.String())
}
func ethernetNicsUp(nics ...string) nmstate.State {
	states := map[string]string{}
	for _, nic := range nics {
		states[nic] = "up"
	}
	return ethernetNicsState(states)
}

func linuxBrUp(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: %s
        - name: %s
`, bridgeName, firstSecondaryNic, secondSecondaryNic))
}

func linuxBrUpWithDefaults(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: %s
          vlan:
            mode: trunk
            trunk-tags:
            - id-range:
                max: 4094
                min: 2
        - name: %s
          vlan:
            mode: trunk
            trunk-tags:
            - id-range:
                max: 4094
                min: 2
`, bridgeName, firstSecondaryNic, secondSecondaryNic))
}

func linuxBrAbsent(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
`, bridgeName))
}

func linuxBrUpNoPorts(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port: []
`, bridgeName))
}

func linuxBrUpWithDisabledVlan(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: %s
          vlan: {}
`, bridgeName, firstSecondaryNic))
}

func ovsBrAbsent(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: absent`, bridgeName))
}

func ovsBrUp(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port:
        - name: %s
        - name: %s
`, bridgeName, firstSecondaryNic, secondSecondaryNic))
}

func ovsbBrWithInternalInterface(bridgeName string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: ovs0
    type: ovs-interface
    state: up
    ipv4:
      enabled: true
      address:
        - ip: 192.0.2.1
          prefix-length: 24
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: true
      port:
        - name: %s
        - name: ovs0`,
		bridgeName, firstSecondaryNic))
}

func ifaceUpWithStaticIP(iface string, ipAddress string, prefixLen string) nmstate.State {
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
`, iface, ipAddress, prefixLen))
}

func ifaceUpWithStaticIPAbsent(firstSecondaryNic string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: false
`, firstSecondaryNic))
}

func ifaceUpWithVlanUp(iface string, vlanID string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
    - name: %s.%s
      type: vlan
      state: up
      vlan:
        base-iface: %s
        id: %s
`, iface, vlanID, iface, vlanID))
}

func vlanAbsent(iface string, vlanID string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
    - name: %s.%s
      type: vlan
      state: absent
      vlan:
        base-iface: %s
        id: %s
`, iface, vlanID, iface, vlanID))
}

func interfaceAbsent(iface string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
    - name: %s
      state: absent
`, iface))
}

func ifaceDownIPv4Disabled(iface string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
    - name: %s
      type: ethernet
      state: down
      ipv4:
        enabled: false
`, iface))
}

func vlanUpWithStaticIP(iface string, ipAddress string) nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
    - name: %s
      type: vlan
      state: up
      ipv4:
        address:
        - ip: %s
          prefix-length: 24
        dhcp: false
        enabled: true
`, iface, ipAddress))
}

func resetPrimaryAndSecondaryNICs() nmstate.State {
	return nmstate.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
  - name: %s
    type: ethernet
    state: down
    ipv4:
      enabled: false
    ipv6:
      enabled: false
  - name: %s
    type: ethernet
    state: down
    ipv4:
      enabled: false
    ipv6:
      enabled: false

`, primaryNic, firstSecondaryNic, secondSecondaryNic))
}

func bridgeOnTheSecondaryInterfaceState() nmstate.State {
	return nmstate.NewState(`interfaces:
  - name: brext
    type: linux-bridge
    state: up
    ipv4: "{{ capture.secondary-iface.interfaces.0.ipv4 }}"
    bridge:
      options:
        stp:
          enabled: false
      port:
      - name: "{{ capture.secondary-iface.interfaces.0.name }}"
routes:
  config: "{{ capture.bridge-routes.routes.running }}"
`)
}

func matchingBond(expectedBond map[string]interface{}) types.GomegaMatcher {
	expectedLinkAggregation := expectedBond["link-aggregation"].(map[string]interface{})
	expectedOptions := expectedLinkAggregation["options"].(map[string]interface{})
	return SatisfyAll(
		HaveKeyWithValue("name", expectedBond["name"]),
		HaveKeyWithValue("type", expectedBond["type"]),
		HaveKeyWithValue("state", expectedBond["state"]),
		HaveKeyWithValue("link-aggregation", SatisfyAll(
			HaveKeyWithValue("mode", expectedLinkAggregation["mode"]),
			HaveKeyWithValue(portFieldName, ConsistOf(expectedLinkAggregation[portFieldName])),
			HaveKeyWithValue("options", HaveKeyWithValue("miimon", expectedOptions["miimon"])),
		)),
	)
}
