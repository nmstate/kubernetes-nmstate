package helper

import (
	"fmt"

	"github.com/tidwall/gjson"

	yaml "sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func getBridgesUp(desiredState nmstate.State) (map[string][]string, error) {
	foundBridgesWithPorts := map[string][]string{}

	desiredStateYaml, err := yaml.YAMLToJSON([]byte(desiredState.Raw))
	if err != nil {
		return foundBridgesWithPorts, fmt.Errorf("error converting desiredState to JSON: %v", err)
	}

	bridgesUp := gjson.ParseBytes(desiredStateYaml).
		Get("interfaces.#(type==linux-bridge)#").
		Get("#(state==up)#").
		Array()

	for _, bridgeUp := range bridgesUp {
		portList := []string{}
		for _, port := range bridgeUp.Get("bridge.port.#.name").Array() {
			portList = append(portList, port.String())
		}

		foundBridgesWithPorts[bridgeUp.Get("name").String()] = portList
	}

	return foundBridgesWithPorts, nil
}
