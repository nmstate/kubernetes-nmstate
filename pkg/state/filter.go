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
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"

	nmstateapiv2 "github.com/nmstate/nmstate/rust/src/go/api/v2"
)

const (
	InterfaceFilter = "interface_filter"
)

func init() {
	if !environment.IsHandler() {
		return
	}
}

func FilterOut(currentState *nmstateapiv2.NetworkState) (*nmstateapiv2.NetworkState, error) {
	return filterOut(currentState)
}

func filterOutRoutes(routes *[]nmstateapiv2.RouteEntry, filteredInterfaces []nmstateapiv2.Interface) *[]nmstateapiv2.RouteEntry {
	if routes == nil {
		return nil
	}
	filteredRoutes := []nmstateapiv2.RouteEntry{}
	for _, route := range *routes {
		name := route.NextHopIface
		if name != nil {
			if isInInterfaces(*name, filteredInterfaces) {
				filteredRoutes = append(filteredRoutes, route)
			}
		}
	}
	return &filteredRoutes
}

func isInInterfaces(interfaceName string, interfaces []nmstateapiv2.Interface) bool {
	for i := range interfaces {
		if interfaces[i].Name == interfaceName {
			return true
		}
	}
	return false
}

func filterOutDynamicAttributes(iface *nmstateapiv2.Interface) *nmstateapiv2.Interface {
	// The gc-timer and hello-time are deep into linux-bridge like this
	//    - bridge:
	//        options:
	//          gc-timer: 13715
	//          hello-timer: 0
	if iface.Type != nmstateapiv2.InterfaceTypeLinuxBridge {
		return iface
	}

	if iface.BridgeConfig == nil {
		return iface
	}

	if iface.BridgeConfig.Options == nil {
		return iface
	}
	iface.BridgeConfig.Options.GcTimer = nil
	iface.BridgeConfig.Options.HelloTimer = nil
	return iface
}

func filterOutInterfaces(interfaces []nmstateapiv2.Interface) []nmstateapiv2.Interface {
	filteredInterfaces := []nmstateapiv2.Interface{}
	for i := range interfaces {
		iface := &interfaces[i]
		if iface.Type == nmstateapiv2.InterfaceTypeVeth && iface.State == nmstateapiv2.InterfaceStateIgnore {
			continue
		}
		filteredInterfaces = append(filteredInterfaces, *filterOutDynamicAttributes(iface))
	}
	return filteredInterfaces
}

func filterOut(currentState *nmstateapiv2.NetworkState) (*nmstateapiv2.NetworkState, error) {
	currentState.Interfaces = filterOutInterfaces(currentState.Interfaces)
	if currentState.Routes != nil {
		currentState.Routes.Running = filterOutRoutes(currentState.Routes.Running, currentState.Interfaces)
		currentState.Routes.Config = filterOutRoutes(currentState.Routes.Config, currentState.Interfaces)
	}

	return currentState, nil
}
