package state

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

type rootState struct {
	Interfaces []interfaceState `json:"interfaces"             yaml:"interfaces"`
	Routes     *routes          `json:"routes,omitempty"       yaml:"routes,omitempty"`
}

type routes struct {
	Config  []routeState `json:"config"  yaml:"config"`
	Running []routeState `json:"running" yaml:"running"`
}

type routeState struct {
	routeFields `yaml:",inline"`
	Data        map[string]interface{}
}

type interfaceState struct {
	interfaceFields `yaml:",inline"`
	Data            map[string]interface{}
}

// interfaceFields allows unmarshaling directly into the defined fields
type interfaceFields struct {
	Name string `json:"name" yaml:"name"`
}

// routeFields allows unmarshaling directly into the defined fields
type routeFields struct {
	NextHopInterface string `json:"next-hop-interface" yaml:"next-hop-interface"`
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
	i.interfaceFields = ifaceFields
	return nil
}

func (r routeState) MarshalJSON() (output []byte, err error) {
	r.Data["next-hop-interface"] = r.NextHopInterface
	return json.Marshal(r.Data)
}

func (r *routeState) UnmarshalJSON(b []byte) error {
	if err := yaml.Unmarshal(b, &r.Data); err != nil {
		return fmt.Errorf("failed Unmarshaling b: %w", err)
	}

	var fields routeFields
	if err := yaml.Unmarshal(b, &fields); err != nil {
		return fmt.Errorf("failed Unmarchaling raw: %w", err)
	}
	r.Data["next-hop-interface"] = fields.NextHopInterface
	r.routeFields = fields
	return nil
}
