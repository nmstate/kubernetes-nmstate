package helper

import (
	"fmt"

	"github.com/tidwall/gjson"

	yaml "sigs.k8s.io/yaml"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func getBridgesUp(desiredState nmstatev1alpha1.State) ([]string, error) {
	foundBridges := []string{}

	desiredStateYaml, err := yaml.YAMLToJSON([]byte(desiredState))
	if err != nil {
		return foundBridges, fmt.Errorf("error converting desiredState to JSON: %v", err)
	}

	queryResults := gjson.ParseBytes(desiredStateYaml).
		Get("interfaces.#(type==linux-bridge)#").
		Get("#(state==up)#.name").
		Array()

	for _, queryResult := range queryResults {
		foundBridges = append(foundBridges, queryResult.String())
	}

	return foundBridges, nil
}
