package state

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

type rootState struct {
	Interfaces  []interfaceState `json:"interfaces" yaml:"interfaces"`
	Routes      *routesState     `json:"routes,omitempty" yaml:"routes,omitempty"`
	DNSResolver *dnsResolver     `json:"dns-resolver,omitempty" yaml:"dns-resolver,omitempty"`
}

type routesState struct {
	Config  []interface{} `json:"config" yaml:"config"`
	Running []interface{} `json:"running" yaml:"running"`
}

type interfaceState struct {
	interfaceFields `yaml:",inline"`
	Data            map[string]interface{}
}

type dnsResolver struct {
	Config  *DNSResolverData `json:"config,omitempty" yaml:"config,omitempty"`
	Running *DNSResolverData `json:"running,omitempty" yaml:"running,omitempty"`
}

type DNSResolverData struct {
	Search []interface{} `json:"search" yaml:"search"`
	Server []interface{} `json:"server" yaml:"server"`
}

// interfaceFields allows unmarshaling directly into the defined fields
type interfaceFields struct {
	Name string `json:"name" yaml:"name"`
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
