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

package qeth

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/go-logr/logr"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
)

type vniccApplier interface {
	Apply(ifaceName string, cfg VniccConfig) error
}

// VniccHook processes the NNCP desiredState:
//  1. Parses any qeth.vnicc config from interface entries
//  2. Applies it via sysfs (qeth driver, not NetworkManager)
//  3. Returns a cleaned desiredState with qeth.vnicc stripped
//     so nmstatectl receives clean input with no unknown fields
type VniccHook struct {
	log     logr.Logger
	manager *Manager     // real sysfs manager
	applier vniccApplier // overridden in tests via hookWithFake
}

// NewVniccHook creates a VniccHook backed by a real sysfs Manager.
func NewVniccHook(log logr.Logger) *VniccHook {
	mgr := NewManager(log)
	return &VniccHook{
		log:     log.WithName("qeth-vnicc-hook"),
		manager: mgr,
		applier: mgr,
	}
}

// ProcessAndStrip is the main entry point called from the controller Reconcile.
//
// On non-s390x platforms this is a no-op (returns state unchanged).
// On s390x:
//   - Parses state.Raw as JSON
//   - Finds interfaces with qeth.vnicc, applies via sysfs
//   - Strips qeth.vnicc from the JSON
//   - Rebuilds shared.State from cleaned JSON
func (h *VniccHook) ProcessAndStrip(state shared.State) (shared.State, error) {
	if runtime.GOARCH != "s390x" {
		return state, nil
	}

	if len(state.Raw) == 0 {
		return state, nil
	}

	// state.Raw is YAML. MarshalJSON() converts YAML→JSON.
	rawJSON, err := state.MarshalJSON()
	if err != nil {
		return state, fmt.Errorf("failed to marshal desiredState to JSON: %w", err)
	}

	cleanedJSON, err := h.processJSON(rawJSON)
	if err != nil {
		return state, err
	}

	// Rebuild shared.State from cleaned JSON
	var cleaned shared.State
	if err := json.Unmarshal(cleanedJSON, &cleaned); err != nil {
		return state, fmt.Errorf("failed to rebuild shared.State after vnicc processing: %w", err)
	}

	return cleaned, nil
}

// processJSON contains the core JSON-level logic.
// Exported-friendly in tests via internal package access.
func (h *VniccHook) processJSON(rawJSON []byte) ([]byte, error) {
	var stateMap map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &stateMap); err != nil {
		return nil, fmt.Errorf("failed to parse desiredState JSON: %w", err)
	}

	rawIfaces, ok := stateMap["interfaces"]
	if !ok {
		return rawJSON, nil
	}

	var ifaces []json.RawMessage
	if err := json.Unmarshal(rawIfaces, &ifaces); err != nil {
		return nil, fmt.Errorf("failed to parse interfaces array: %w", err)
	}

	cleaned := make([]json.RawMessage, 0, len(ifaces))
	for _, rawIface := range ifaces {
		c, err := h.processIface(rawIface)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, c)
	}

	cleanedIfacesJSON, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encode interfaces: %w", err)
	}
	stateMap["interfaces"] = cleanedIfacesJSON

	return json.Marshal(stateMap)
}

// processIface handles one interface entry:
// applies vnicc via the applier if present, strips the qeth.vnicc key.
func (h *VniccHook) processIface(rawIface json.RawMessage) (json.RawMessage, error) {
	var ifaceMap map[string]json.RawMessage
	if err := json.Unmarshal(rawIface, &ifaceMap); err != nil {
		return rawIface, fmt.Errorf("failed to parse interface entry: %w", err)
	}

	rawQeth, hasQeth := ifaceMap["qeth"]
	if !hasQeth {
		return rawIface, nil
	}

	var qethCfg struct {
		Vnicc *VniccConfig `json:"vnicc,omitempty"`
	}
	if err := json.Unmarshal(rawQeth, &qethCfg); err != nil {
		return rawIface, fmt.Errorf("failed to parse qeth config: %w", err)
	}

	if qethCfg.Vnicc == nil {
		return rawIface, nil // qeth present but no vnicc — pass through unchanged
	}

	// Resolve interface name
	nameRaw, ok := ifaceMap["name"]
	if !ok {
		return rawIface, fmt.Errorf("interface entry missing 'name' field")
	}
	var ifaceName string
	if err := json.Unmarshal(nameRaw, &ifaceName); err != nil {
		return rawIface, fmt.Errorf("failed to parse interface name: %w", err)
	}

	h.log.Info("Applying qeth vnicc configuration", "interface", ifaceName)

	// Use applier (real Manager in production, fakeManager in tests)
	if err := h.applier.Apply(ifaceName, *qethCfg.Vnicc); err != nil {
		return rawIface, fmt.Errorf("vnicc apply failed for %s: %w", ifaceName, err)
	}

	// Strip vnicc from the qeth map — nmstate rejects unknown keys
	var qethMap map[string]json.RawMessage
	if err := json.Unmarshal(rawQeth, &qethMap); err != nil {
		return rawIface, err
	}
	delete(qethMap, "vnicc")

	if len(qethMap) == 0 {
		delete(ifaceMap, "qeth") // remove entire qeth block if nothing remains
	} else {
		cleanedQeth, err := json.Marshal(qethMap)
		if err != nil {
			return rawIface, err
		}
		ifaceMap["qeth"] = cleanedQeth
	}

	return json.Marshal(ifaceMap)
}
