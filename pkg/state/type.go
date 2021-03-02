package state

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

type rootState struct {
	Interfaces []interfaceState `json:"interfaces"`
	Routes     *routesState     `json:"routes,omitempty"`
}

type routesState struct {
	Config  []interface{} `json:"config"`
	Running []interface{} `json:"running"`
}

type interfaceState struct {
	interfaceFields
	Data map[string]interface{}
}

// interfaceFields allows unmarshaling directly into the defined fields
type interfaceFields struct {
	Name string `json:"name"`
}

func (i interfaceState) MarshalJSON() (output []byte, err error) {
	i.Data["name"] = i.Name
	return json.Marshal(i.Data)
}

func (i *interfaceState) UnmarshalJSON(b []byte) error {
	if err := yaml.Unmarshal(b, &i.Data); err != nil {
		return fmt.Errorf("failed Unmarshaling b: %w", err)
	}

	var ifaceFields interfaceFields
	if err := yaml.Unmarshal(b, &ifaceFields); err != nil {
		return fmt.Errorf("failed Unmarshaling raw: %w", err)
	}
	i.Data["name"] = ifaceFields.Name
	i.Name = ifaceFields.Name
	return nil
}
