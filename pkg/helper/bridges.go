package helper

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	yaml "sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var defaultVlanFiltering = map[string]interface{}{
	"mode": "trunk",
	"trunk-tags": []map[string]interface{}{
		{
			"id-range": map[string]interface{}{
				"min": 2,
				"max": 4094,
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
			result, err = sjson.SetBytes(result, fmt.Sprintf("interfaces.%d.bridge.port.%d.vlan", ifaceIndex, portIndex), defaultVlanFiltering)
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
