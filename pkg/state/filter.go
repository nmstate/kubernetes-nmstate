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
	"strings"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"

	yaml "sigs.k8s.io/yaml"
)

const (
	InterfaceFilter = "interface_filter"
)

func init() {
	if !environment.IsHandler() {
		return
	}
}

func FilterOut(currentState shared.State) (shared.State, error) {
	return filterOut(currentState)
}

// CountInterfacesByType parses the state and returns a map of interface type to count.
func CountInterfacesByType(currentState shared.State) (map[string]int, error) {
	var state rootState
	if err := yaml.Unmarshal(currentState.Raw, &state); err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, iface := range state.Interfaces {
		if iface.Type != "" {
			counts[iface.Type]++
		}
	}
	return counts, nil
}

// RouteKey represents the grouping key for route metrics.
type RouteKey struct {
	IPStack string // "ipv4" or "ipv6"
	Type    string // "static" or "dynamic"
}

// CountRoutes parses the state and returns a map of RouteKey to count.
// Routes are categorized by:
// - IP stack: determined by presence of ":" in destination (ipv6) or not (ipv4)
// - Type: "static" if route exists in routes.config, "dynamic" if only in routes.running
func CountRoutes(currentState shared.State) (map[RouteKey]int, error) {
	var state rootState
	if err := yaml.Unmarshal(currentState.Raw, &state); err != nil {
		return nil, err
	}

	counts := make(map[RouteKey]int)
	if state.Routes == nil {
		return counts, nil
	}

	// Build a set of static route destinations for quick lookup
	staticRoutes := make(map[string]struct{})
	for _, route := range state.Routes.Config {
		staticRoutes[route.Destination] = struct{}{}
	}

	// Count running routes by IP stack and type
	for _, route := range state.Routes.Running {
		ipStack := getIPStack(route.Destination)
		routeType := "dynamic"
		if _, isStatic := staticRoutes[route.Destination]; isStatic {
			routeType = "static"
		}

		key := RouteKey{
			IPStack: ipStack,
			Type:    routeType,
		}
		counts[key]++
	}

	return counts, nil
}

// getIPStack determines the IP stack from a destination CIDR.
// Returns "ipv6" if the destination contains ":", otherwise "ipv4".
func getIPStack(destination string) string {
	if strings.Contains(destination, ":") {
		return "ipv6"
	}
	return "ipv4"
}

func filterOutRoutes(routes []routeState, filteredInterfaces []interfaceState) []routeState {
	filteredRoutes := []routeState{}
	for _, route := range routes {
		name := route.NextHopInterface
		if isInInterfaces(name, filteredInterfaces) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
}

func isInInterfaces(interfaceName string, interfaces []interfaceState) bool {
	for _, iface := range interfaces {
		if iface.Name == interfaceName {
			return true
		}
	}
	return false
}

func filterOutDynamicAttributes(iface map[string]interface{}) {
	filterOutBridgeDynamicAttributes(iface)
	filterOutIPAddressLifetimeAttributes(iface)
}

func filterOutBridgeDynamicAttributes(iface map[string]interface{}) {
	// The gc-timer and hello-time are deep into linux-bridge like this
	//    - bridge:
	//        options:
	//          gc-timer: 13715
	//          hello-timer: 0
	if iface["type"] != "linux-bridge" {
		return
	}

	bridgeRaw, hasBridge := iface["bridge"]
	if !hasBridge {
		return
	}
	bridge, ok := bridgeRaw.(map[string]interface{})
	if !ok {
		return
	}

	optionsRaw, hasOptions := bridge["options"]
	if !hasOptions {
		return
	}
	options, ok := optionsRaw.(map[string]interface{})
	if !ok {
		return
	}

	delete(options, "gc-timer")
	delete(options, "hello-timer")
}

func filterOutIPAddressLifetimeAttributes(iface map[string]interface{}) {
	// The preferred-life-time and valid-life-time are in IPv4/IPv6 address entries like this:
	//    - ipv4:
	//        address:
	//          - ip: 192.168.1.1
	//            prefix-length: 24
	//            preferred-life-time: 3600
	//            valid-life-time: 7200
	filterOutAddressLifetimes(iface, "ipv4")
	filterOutAddressLifetimes(iface, "ipv6")
}

func filterOutAddressLifetimes(iface map[string]interface{}, ipVersion string) {
	ip, ok := iface[ipVersion].(map[string]interface{})
	if !ok {
		return
	}

	addresses, ok := ip["address"].([]interface{})
	if !ok {
		return
	}

	for _, addrRaw := range addresses {
		addr, ok := addrRaw.(map[string]interface{})
		if !ok {
			continue
		}
		delete(addr, "preferred-life-time")
		delete(addr, "valid-life-time")
	}
}

func filterOutInterfaces(ifacesState []interfaceState) []interfaceState {
	filteredInterfaces := []interfaceState{}
	for _, iface := range ifacesState {
		if isVeth(iface.Data) && isUnmanaged(iface.Data) {
			continue
		}
		filterOutDynamicAttributes(iface.Data)
		filteredInterfaces = append(filteredInterfaces, iface)
	}
	return filteredInterfaces
}

func isVeth(ifaceData map[string]interface{}) bool {
	return ifaceData["type"] == "veth"
}

func isUnmanaged(ifaceData map[string]interface{}) bool {
	return ifaceData["state"] == "ignore"
}

func filterOut(currentState shared.State) (shared.State, error) {
	var state rootState
	if err := yaml.Unmarshal(currentState.Raw, &state); err != nil {
		return currentState, err
	}

	state.Interfaces = filterOutInterfaces(state.Interfaces)
	if state.Routes != nil {
		state.Routes.Running = filterOutRoutes(state.Routes.Running, state.Interfaces)
		state.Routes.Config = filterOutRoutes(state.Routes.Config, state.Interfaces)
	}

	filteredState, err := yaml.Marshal(state)
	if err != nil {
		return currentState, err
	}

	return shared.NewState(string(filteredState)), nil
}
