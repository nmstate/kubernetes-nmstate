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

package bridge

import (
	nmstateapiv2 "github.com/nmstate/nmstate/rust/src/go/api/v2"
)

const minVlanID = 2
const maxVlanID = 4094

func ApplyDefaultVlanFiltering(desiredState nmstateapiv2.NetworkState) (nmstateapiv2.NetworkState, error) {
	// Make linter happy use index since iface is too big
	for ifaceIndex := range desiredState.Interfaces {
		iface := &desiredState.Interfaces[ifaceIndex]
		if !isLinuxBridgeUp(iface) || iface.BridgeInterface == nil || iface.BridgeConfig == nil || iface.BridgeConfig.Ports == nil {
			continue
		}

		for portIndex, port := range *iface.BridgeConfig.Ports {
			if hasVlanConfiguration(port) {
				continue
			}
			trunkMode := nmstateapiv2.BridgePortVlanModeTrunk
			(*desiredState.Interfaces[ifaceIndex].BridgeConfig.Ports)[portIndex].Vlan = &nmstateapiv2.BridgePortVlanConfig{
				Mode: &trunkMode,
				TrunkTags: &[]nmstateapiv2.BridgePortTrunkTag{{
					IDRange: &nmstateapiv2.BridgePortVlanRange{
						Min: minVlanID,
						Max: maxVlanID,
					},
				}},
			}
		}
	}
	return desiredState, nil
}

func isLinuxBridgeUp(iface *nmstateapiv2.Interface) bool {
	return iface.Type == nmstateapiv2.InterfaceTypeLinuxBridge && iface.State == nmstateapiv2.InterfaceStateUp
}

func hasVlanConfiguration(port nmstateapiv2.BridgePortConfig) bool {
	return port.Vlan != nil
}
