package helper

import (
	"fmt"

	"github.com/tidwall/gjson"

	yaml "sigs.k8s.io/yaml"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func getBridgesUp(desiredState nmstatev1alpha1.State) (map[string][]string, error) {
	foundBridgesWithPorts := map[string][]string{}

	desiredStateYaml, err := yaml.YAMLToJSON([]byte(desiredState))
	if err != nil {
		return foundBridgesWithPorts, fmt.Errorf("error converting desiredState to JSON: %v", err)
	}

	queryResults := gjson.ParseBytes(desiredStateYaml).
		Get("interfaces.#(type==linux-bridge)#").
		Get("#(state==up)#.name").
		Array()

	for _, queryResult := range queryResults {
		portsQueryResults := gjson.ParseBytes(desiredStateYaml).
			Get("interfaces.#(type==linux-bridge)#").
			Get(fmt.Sprintf("#(name==%s)#.bridge.port.#.name",queryResult.String())).
			Array()

		portList := []string{}
		for _, ports := range portsQueryResults {
			for _, portName := range ports.Array() {
				portList = append(portList,portName.String() )
			}
		}

		foundBridgesWithPorts[queryResult.String()] = portList
	}

	return foundBridgesWithPorts, nil
}
