package state

import (
	"os"

	"github.com/gobwas/glob"
	"github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"

	yaml "sigs.k8s.io/yaml"
)

var (
	interfacesFilterGlobFromEnv glob.Glob
)

func init() {
	if !environment.IsHandler() {
		return
	}
	interfacesFilter, isSet := os.LookupEnv("INTERFACES_FILTER")
	if !isSet {
		panic("INTERFACES_FILTER is mandatory")
	}
	interfacesFilterGlobFromEnv = glob.MustCompile(interfacesFilter)
}

func FilterOut(currentState shared.State) (shared.State, error) {
	return filterOut(currentState, interfacesFilterGlobFromEnv)
}

func filterOutRoutes(routes []interface{}, interfacesFilterGlob glob.Glob) []interface{} {
	filteredRoutes := []interface{}{}
	for _, route := range routes {
		name := route.(map[string]interface{})["next-hop-interface"]
		if !interfacesFilterGlob.Match(name.(string)) {
			filteredRoutes = append(filteredRoutes, route)
		}
	}

	return filteredRoutes
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

func filterOutInterfaces(ifacesState []interfaceState, interfacesFilterGlob glob.Glob) []interfaceState {
	filteredInterfaces := []interfaceState{}
	for _, iface := range ifacesState {
		if !interfacesFilterGlob.Match(iface.Name) {
			filterOutDynamicAttributes(iface.Data)
			filteredInterfaces = append(filteredInterfaces, iface)
		}
	}
	return filteredInterfaces
}

func filterOut(currentState shared.State, interfacesFilterGlob glob.Glob) (shared.State, error) {
	var state rootState
	if err := yaml.Unmarshal(currentState.Raw, &state); err != nil {
		return currentState, err
	}

	state.Interfaces = filterOutInterfaces(state.Interfaces, interfacesFilterGlob)
	if state.Routes != nil {
		state.Routes.Running = filterOutRoutes(state.Routes.Running, interfacesFilterGlob)
		state.Routes.Config = filterOutRoutes(state.Routes.Config, interfacesFilterGlob)
	}
	filteredState, err := yaml.Marshal(state)
	if err != nil {
		return currentState, err
	}

	return shared.NewState(string(filteredState)), nil
}
