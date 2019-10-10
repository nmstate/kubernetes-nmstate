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

func brUp(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      port:
        - name: eth1
        - name: eth2
`, bridgeName))
}

func brAbsent(bridgeName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
`, bridgeName))
}

func brUpNoPorts(bridgeName string) nmstatev1alpha1.State {
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
