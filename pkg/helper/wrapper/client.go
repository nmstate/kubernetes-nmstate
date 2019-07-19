package wrapper

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	yaml "sigs.k8s.io/yaml"
)

const (
	SET_VLAN_FILTERING = "ip link set %s type bridge vlan_filtering 1"
	SET_PORT_VLAN      = "bridge vlan add dev %s vid %d"
	SET_BRIDGE_VLAN    = SET_PORT_VLAN + " self"
)

type FilteredStateWithExtraCommands struct {
	state         nmstatev1alpha1.State
	extraCommands []string
}

// Generic function to compose vlan setting commands reading the following
// yaml fields 'name', 'vlan-range-min' and 'vlan-range-max' multiple
// commands can be compose since a range can be spicifed for vlans
func composeVlansCommands(state gjson.Result, vlansPath string, command string) []string {
	var extraCommands []string
	name := state.Get("name").String()
	vlans := state.Get(vlansPath).Array()
	for _, vlan := range vlans {
		minResult := vlan.Get("vlan-range-min")
		if !minResult.Exists() {
			continue
		}
		min := minResult.Int()
		max := min
		maxResult := vlan.Get("vlan-range-max")
		if maxResult.Exists() {
			max = maxResult.Int()
		}
		for vid := min; vid <= max; vid++ {
			extraCommands = append(extraCommands, fmt.Sprintf(command, name, vid))
		}
	}
	return extraCommands
}

// This will set the vlan_filtering to 1 and add the expected
// vlans for bridge and for ports
func composeVlanFilteringCommands(parsedState gjson.Result) []string {
	var extraCommands []string
	// Get all the linux bridges with vlan-filtering: true
	vlanFilteredBridges := parsedState.
		Get("interfaces.#(type==linux-bridge)#").
		Get("#(bridge.options.vlan-filtering==true)#").Array()

	// Compose the iproute command to set vlan_filtering 1
	for _, vlanFilteredBridge := range vlanFilteredBridges {

		// Compose the vlan_filtering 1 command bridge setting command
		// ej, ip link set br1 type bridge vlan_filtering 1
		bridgeName := vlanFilteredBridge.Get("name").String()
		setVlanFilteringCmd := fmt.Sprintf(SET_VLAN_FILTERING, bridgeName)
		extraCommands = []string{setVlanFilteringCmd}

		// Compose the bridge device itself vlans
		// ej, bridge vlan add dev br1 vid 10 self
		setBridgeVlansCmds := composeVlansCommands(vlanFilteredBridge, "bridge.options.vlans", SET_BRIDGE_VLAN)
		extraCommands = append(extraCommands, setBridgeVlansCmds...)

		// Compose the ports vlans
		// ej, bridge vlan add dev eth1 vid 100
		ports := vlanFilteredBridge.Get("bridge.port").Array()
		for _, port := range ports {
			setPortVlansCmds := composeVlansCommands(port, "vlans", SET_PORT_VLAN)
			extraCommands = append(extraCommands, setPortVlansCmds...)
		}
	}
	return extraCommands
}

// sjson is not able to delete fields from array of structs without
// passing index so we discover all number of interfaces and the
// number of ports per interface to compose the json path to delete
func composeVlanConfigPath(state gjson.Result) []string {
	var filters []string
	// Get number of ports per interface
	portCardinalityByIface := map[int]int64{}
	interfaces := state.Get("interfaces").Array()
	for idx, iface := range interfaces {
		portCardinality := iface.Get("bridge.port.#").Int()
		portCardinalityByIface[idx] = portCardinality
	}

	// Iterate over the interfaces and filter out all the bridge vlan configuration
	bridgeFilters := []string{
		"interfaces.%d.bridge.options.vlan-filtering",
		"interfaces.%d.bridge.options.vlans",
	}
	portFilters := []string{
		"interfaces.%d.bridge.port.%d.vlans",
	}
	for i, portCardinality := range portCardinalityByIface {
		for _, bridgeFilter := range bridgeFilters {
			filters = append(filters, fmt.Sprintf(bridgeFilter, i))
		}
		for p := int64(0); p < portCardinality; p++ {
			for _, portFilter := range portFilters {
				filters = append(filters, fmt.Sprintf(portFilter, i, p))
			}
		}
	}

	return filters
}

// It will compose the iproute commands to configure vlans on a linux bridge and
// filter out this configuration from desiredState to pass it to nmstate
func processLinuxBridgeVlans(desiredState nmstatev1alpha1.State) (FilteredStateWithExtraCommands, error) {

	result := FilteredStateWithExtraCommands{state: desiredState}
	if len(desiredState) == 0 {
		return result, nil
	}

	// Convert to json so we can use search tool gjson
	json, err := yaml.YAMLToJSON(desiredState)
	if err != nil {
		return result, fmt.Errorf("error desiredState converting yaml to json: %v", err)
	}

	parsedDesiredState := gjson.ParseBytes(json)

	result.extraCommands = composeVlanFilteringCommands(parsedDesiredState)

	// Delete vlan filtering configuration
	vlanConfigPaths := composeVlanConfigPath(parsedDesiredState)
	filteredOutState := parsedDesiredState.Raw
	for _, vlanConfigPath := range vlanConfigPaths {
		var err error
		filteredOutState, err = sjson.Delete(filteredOutState, vlanConfigPath)
		if err != nil {
			return result, err
		}
	}

	// Convert it back to YAML
	filteredOutStateYaml, err := yaml.JSONToYAML([]byte(filteredOutState))
	if err != nil {
		return result, fmt.Errorf("error converting filtered out desiredState from json to yaml: %v", err)
	}
	result.state = filteredOutStateYaml
	return result, nil
}

func run(command string) (string, error) {
	commandSplitted := strings.Split(command, " ")
	cmd := exec.Command(commandSplitted[0], commandSplitted[:len(commandSplitted)]...)
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute ip route command '%s': '%v' '%s' '%s'", command, err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil

}

func ApplyDesiredState(nodeNetworkState *nmstatev1alpha1.NodeNetworkState) (string, error) {
	filteredStateWithExtraCommands, err := processLinuxBridgeVlans(nodeNetworkState.Spec.DesiredState)
	if err != nil {
		return "", fmt.Errorf("error processing linux bridge vlan configuration %v", err)
	}

	// First apply filtered out desired state so interfaces and bridges are
	// created
	nodeNetworkState.Spec.DesiredState = filteredStateWithExtraCommands.state
	output, err := nmstate.ApplyDesiredState(nodeNetworkState)
	if err != nil {
		return "", err
	}

	// Then execute all the iproute commands one by one to configure
	// vlan filtering at linux bridges if needed
	for _, extraCommand := range filteredStateWithExtraCommands.extraCommands {
		//TODO: What do we do with errors here ?
		_, _ = run(extraCommand)
	}

	// Return the vlan configuration to log it
	bridgeVlanState, err := run("bridge vlan show")
	if err != nil {
		return "", fmt.Errorf("error retrieving linux bridge vlans")
	}
	return output + bridgeVlanState, nil
}
