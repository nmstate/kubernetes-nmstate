package e2e

import (
	"fmt"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func ethernetNicUp(nicName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
`, nicName))
}

func linuxBrUp(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
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
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
`, bridgeName))
}

func linuxBrUpNoPorts(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
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
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: absent`, bridgeName))
}

func ovsBrUp(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port:
        - name: eth1
        - name: eth2
`, bridgeName))
}

func ovsBrUpNoPorts(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: ovs-bridge
    state: up
    bridge:
      options:
        stp: false
      port: []
`, bridgeName))
}
