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
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	yaml "sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

const minVlanID = 2
const maxVlanID = 4094

var defaultVlanFiltering = map[string]interface{}{
	"mode": "trunk",
	"trunk-tags": []map[string]interface{}{
		{
			"id-range": map[string]interface{}{
				"min": minVlanID,
				"max": maxVlanID,
			},
		},
	},
}

func ApplyDefaultVlanFiltering(desiredState nmstate.State) (nmstate.State, error) {
	result, err := yaml.YAMLToJSON(desiredState.Raw)
	if err != nil {
		return desiredState, fmt.Errorf("error converting desiredState to JSON: %v", err)
	}

	ifaces := gjson.ParseBytes(result).Get("interfaces").Array()
	for ifaceIndex, iface := range ifaces {
		if !isLinuxBridgeUp(iface) {
			continue
		}
		for portIndex, port := range iface.Get("bridge.port").Array() {
			if hasVlanConfiguration(port) {
				continue
			}
			result, err = sjson.SetBytes(
				result,
				fmt.Sprintf("interfaces.%d.bridge.port.%d.vlan", ifaceIndex, portIndex),
				defaultVlanFiltering,
			)
			if err != nil {
				return desiredState, err
			}
		}
	}

	resultYaml, err := yaml.JSONToYAML(result)
	if err != nil {
		return desiredState, err
	}
	return nmstate.State{Raw: resultYaml}, nil
}

func isLinuxBridgeUp(iface gjson.Result) bool {
	return iface.Get("type").String() == "linux-bridge" && iface.Get("state").String() == "up"
}

func hasVlanConfiguration(port gjson.Result) bool {
	return port.Get("vlan").Exists()
}
