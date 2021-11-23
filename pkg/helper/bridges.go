package helper

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	yaml "sigs.k8s.io/yaml"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
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

func EnableVlanFiltering() (string, error) {
	currentState, err := nmstatectl.Show()
	if err != nil {
		return "failed to get currentState", err
	}
	upBridgesWithPorts, err := GetUpLinuxBridgesWithPorts(shared.NewState(currentState))
	if err != nil {
		return "failed to list bridges with ports", err
	}
	out, err := enableVlanFiltering(upBridgesWithPorts)
	if err != nil {
		return fmt.Sprintf("failed to enable vlan-filtering via nmcli: %s", out), err
	}
	return "", nil
}

func GetUpLinuxBridgesWithPorts(desiredState nmstate.State) (map[string][]string, error) {
	bridgesWithPorts := map[string][]string{}

	result, err := yaml.YAMLToJSON(desiredState.Raw)
	if err != nil {
		return bridgesWithPorts, fmt.Errorf("error converting desiredState to JSON: %v", err)
	}

	ifaces := gjson.ParseBytes(result).Get("interfaces").Array()
	for _, iface := range ifaces {
		if !isLinuxBridgeUp(iface) {
			continue
		}
		for _, port := range iface.Get("bridge.port").Array() {
			if hasVlanConfiguration(port) {
				continue
			}
			bridgeName := iface.Get("name").String()
			bridgesWithPorts[bridgeName] = append(bridgesWithPorts[bridgeName], port.Get("name").String())
		}
	}
	return bridgesWithPorts, nil
}

func enableVlanFiltering(upBridgesWithPorts map[string][]string) (string, error) {
	for bridge, ports := range upBridgesWithPorts {
		out, err := enableBridgeVlanFiltering(bridge)
		if err != nil {
			return out, err
		}
		for _, port := range ports {
			out, err = enableBridgPortVlans(port)
			if err != nil {
				return out, err
			}
		}
	}
	return "", nil
}

func enableBridgeVlanFiltering(bridgeName string) (string, error) {
	command := "nmcli"
	args := []string{"con", "mod", bridgeName, "bridge.vlan-filtering", "yes"}
	return runCommand(command, args)
}

func enableBridgPortVlans(port string) (string, error) {
	command := "nmcli"
	args := []string{"con", "mod", port, "bridge-port.vlans", "2-4094"}
	return runCommand(command, args)
}

func runCommand(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s %s: '%v', '%s', '%s'", command, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func isLinuxBridgeUp(iface gjson.Result) bool {
	return iface.Get("type").String() == "linux-bridge" && iface.Get("state").String() == "up"
}

func hasVlanConfiguration(port gjson.Result) bool {
	return port.Get("vlan").Exists()
}
