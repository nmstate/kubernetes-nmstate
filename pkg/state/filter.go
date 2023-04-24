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
	"github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"

	goyaml "gopkg.in/yaml.v2"
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
	if err := normalizeInterfacesNames(currentState.Raw, &state); err != nil {
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

// normalizeInterfacesNames fixes the unmarshal of numeric values in the interfaces names
// Numeric values, including the ones with a base prefix (e.g. 0x123) should be stringify.
func normalizeInterfacesNames(rawState []byte, state *rootState) error {
	var stateForNormalization rootState
	if err := goyaml.Unmarshal(rawState, &stateForNormalization); err != nil {
		return err
	}

	for i, iface := range stateForNormalization.Interfaces {
		state.Interfaces[i].Name = iface.Name
	}

	if stateForNormalization.Routes != nil {
		for i, route := range stateForNormalization.Routes.Config {
			state.Routes.Config[i].NextHopInterface = route.NextHopInterface
		}
		for i, route := range stateForNormalization.Routes.Running {
			state.Routes.Running[i].NextHopInterface = route.NextHopInterface
		}
	}
	return nil
}
