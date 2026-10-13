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
	"strings"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	yaml "sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("state")

const (
	InterfaceFilter = "interface_filter"

	// DefaultMaxStateSizeBytes is the default budget, in bytes, for the
	// serialized NodeNetworkState currentState. It leaves headroom under
	// the etcd request size limit (1.5 MiB) for object metadata and the
	// remaining status fields.
	DefaultMaxStateSizeBytes = 1 << 20
)

func init() {
	if !environment.IsHandler() {
		return
	}
}

// FilterOut removes transient or unmanaged entries from the observed state
// and compacts VLAN interfaces when the serialized state exceeds
// maxSizeBytes. It returns the filtered state and the compaction tier that
// was applied to VLAN interfaces, 0 when the state fit without compaction.
// A non-positive maxSizeBytes selects DefaultMaxStateSizeBytes.
func FilterOut(currentState shared.State, maxSizeBytes int) (shared.State, int, error) {
	return filterOut(currentState, maxSizeBytes)
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

func filterOutDynamicAttributes(iface map[string]any) {
	filterOutBridgeDynamicAttributes(iface)
	filterOutIPAddressLifetimeAttributes(iface)
}

func filterOutBridgeDynamicAttributes(iface map[string]any) {
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
	bridge, ok := bridgeRaw.(map[string]any)
	if !ok {
		return
	}

	optionsRaw, hasOptions := bridge["options"]
	if !hasOptions {
		return
	}
	options, ok := optionsRaw.(map[string]any)
	if !ok {
		return
	}

	delete(options, "gc-timer")
	delete(options, "hello-timer")
}

func filterOutIPAddressLifetimeAttributes(iface map[string]any) {
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

func filterOutAddressLifetimes(iface map[string]any, ipVersion string) {
	ip, ok := iface[ipVersion].(map[string]any)
	if !ok {
		return
	}

	addresses, ok := ip["address"].([]any)
	if !ok {
		return
	}

	for _, addrRaw := range addresses {
		addr, ok := addrRaw.(map[string]any)
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

func isVeth(ifaceData map[string]any) bool {
	return ifaceData["type"] == "veth"
}

func isUnmanaged(ifaceData map[string]any) bool {
	return ifaceData["state"] == "ignore"
}

const vlanInterfaceType = "vlan"

// keepSet describes the fields preserved on an interface when it is
// compacted. A nil value keeps the whole subtree under that key, a non-nil
// value prunes the subtree recursively.
type keepSet map[string]keepSet

// vlanCompactionTiers lists, from least to most aggressive, the fields kept
// on VLAN interfaces when the serialized state exceeds the size budget.
// VLANs are the interface type that scales into the thousands on a node.
// The dropped fields default to the base interface values (mac-address,
// mtu) or are static defaults (ethtool, lldp, wait-ip, ...), and the
// per-VLAN configuration stays visible in the policy that applied it.
var vlanCompactionTiers = []keepSet{
	{"name": nil, "type": nil, "state": nil, "ipv4": nil, "ipv6": nil, "vlan": nil},
	{
		"name":  nil,
		"type":  nil,
		"state": nil,
		"ipv4":  {"enabled": nil, "dhcp": nil, "address": nil},
		"ipv6":  {"enabled": nil, "dhcp": nil, "autoconf": nil, "address": nil},
		"vlan":  {"base-iface": nil, "id": nil},
	},
}

// compactToFit applies the VLAN compaction tiers in order until the
// serialized state fits within maxSizeBytes and returns the applied tier,
// 0 when no compaction was needed. If no tier is sufficient the most
// compact result is kept so the update is still attempted.
func compactToFit(state *rootState, maxSizeBytes int) (int, error) {
	if maxSizeBytes <= 0 {
		maxSizeBytes = DefaultMaxStateSizeBytes
	}
	size, err := serializedSize(state)
	if err != nil {
		return 0, err
	}
	if size <= maxSizeBytes {
		return 0, nil
	}
	tier := 0
	for _, keep := range vlanCompactionTiers {
		tier++
		compactVlanInterfaces(state.Interfaces, keep)
		size, err = serializedSize(state)
		if err != nil {
			return tier, err
		}
		if size <= maxSizeBytes {
			return tier, nil
		}
	}
	log.Info("State exceeds the size budget even after compacting VLAN interfaces, the NodeNetworkState update may be rejected",
		"sizeBytes", size, "maxSizeBytes", maxSizeBytes)
	return tier, nil
}

// serializedSize returns the size of the state as stored in the
// NodeNetworkState, which is JSON (see shared.State.MarshalJSON).
func serializedSize(state *rootState) (int, error) {
	serialized, err := json.Marshal(state)
	if err != nil {
		return 0, err
	}
	return len(serialized), nil
}

func compactVlanInterfaces(interfaces []interfaceState, keep keepSet) {
	for i := range interfaces {
		if interfaces[i].Type != vlanInterfaceType {
			continue
		}
		pruneToKeepSet(interfaces[i].Data, keep)
	}
}

func pruneToKeepSet(data map[string]any, keep keepSet) {
	for key, value := range data {
		nestedKeep, found := keep[key]
		if !found {
			delete(data, key)
			continue
		}
		if nestedKeep == nil {
			continue
		}
		if nested, isMap := value.(map[string]any); isMap {
			pruneToKeepSet(nested, nestedKeep)
		}
	}
}

func filterOut(currentState shared.State, maxSizeBytes int) (shared.State, int, error) {
	var state rootState
	if err := yaml.Unmarshal(currentState.Raw, &state); err != nil {
		return currentState, 0, err
	}

	state.Interfaces = filterOutInterfaces(state.Interfaces)
	if state.Routes != nil {
		state.Routes.Running = filterOutRoutes(state.Routes.Running, state.Interfaces)
		state.Routes.Config = filterOutRoutes(state.Routes.Config, state.Interfaces)
	}

	tier, err := compactToFit(&state, maxSizeBytes)
	if err != nil {
		return currentState, 0, err
	}

	filteredState, err := yaml.Marshal(state)
	if err != nil {
		return currentState, 0, err
	}

	return shared.NewState(string(filteredState)), tier, nil
}
