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

package state

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

type rootState struct {
	Interfaces  []interfaceState `json:"interfaces"             yaml:"interfaces"`
	Routes      *routes          `json:"routes,omitempty"       yaml:"routes,omitempty"`
	DNSResolver *dnsResolver     `json:"dns-resolver,omitempty" yaml:"dns-resolver,omitempty"`
	Ovn         *bridgeMappings  `json:"ovn,omitempty"          yaml:"ovn,omitempty"`
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

type dnsResolver struct {
	Config  *DNSResolverData `json:"config,omitempty"  yaml:"config,omitempty"`
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

type bridgeMappings struct {
	PhysicalNetworkMappings []PhysicalNetworks `json:"bridge-mappings,omitempty" yaml:"bridge-mappings,omitempty"`
}

type PhysicalNetworks struct {
	Name   string `json:"localnet" yaml:"localnet"`
	Bridge string `json:"bridge" yaml:"bridge"`
	State  string `json:"state,omitempty" yaml:"state,omitempty"`
}
