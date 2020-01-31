package e2e

import (
	"fmt"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func ethernetNicUp(nicName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
`, nicName))
}

func linuxBrUp(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: %s
        - name: %s
`, bridgeName, *firstSecondaryNic, *secondSecondaryNic))
}

func linuxBrAbsent(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
`, bridgeName))
}

func linuxBrUpNoPorts(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
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

func ovsBrAbsent(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: absent`, bridgeName))
}

func ovsBrUp(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port:
        - name: %s
        - name: %s
`, bridgeName, *firstSecondaryNic, *secondSecondaryNic))
}

func ovsbBrWithInternalInterface(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
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
		bridgeName, *firstSecondaryNic))
}

func ifaceUpWithStaticIP(iface string, ipAddress string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
    - name: %s
      type: ethernet
      state: up
      ipv4:
        address:
        - ip: %s
          prefix-length: 24
        dhcp: false
        enabled: true
`, iface, ipAddress))
}

func ifaceUpWithVlanUp(iface string, vlanId string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
    - name: %s.%s
      type: vlan
      state: up
      vlan:
        base-iface: %s
        id: %s
`, iface, vlanId, iface, vlanId))
}

func vlanAbsent(iface string, vlanId string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
    - name: %s.%s
      type: vlan
      state: absent
      vlan:
        base-iface: %s
        id: %s
`, iface, vlanId, iface, vlanId))
}

func ifaceDownIPv4Disabled(iface string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
    - name: %s
      type: ethernet
      state: down
      ipv4:
        enabled: false
`, iface))
}

func vlanUpWithStaticIP(iface string, ipAddress string) nmstatev1alpha1.State {
	return nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
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
