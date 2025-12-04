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

package nmpolicy

import (
	"encoding/json"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/backend"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

func GenerateState(
	desiredState nmstateapi.State,
	policySpec nmstateapi.NodeNetworkConfigurationPolicySpec,
	currentState nmstateapi.State,
	cachedState map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState) (
	map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState, nmstateapi.State, error) {
	if len(desiredState.Raw) == 0 || len(currentState.Raw) == 0 {
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
			nmstateapi.State{}, nil
	}

	// Check if we're using netplan backend
	currentBackend := backend.GetBackend()
	if currentBackend.Name() == backend.BackendNetplan {
		// Netplan doesn't use the policy validation/capture mechanism
		// It uses Try() for atomic apply with auto-rollback
		// Just pass through the desired state without validation
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
			policySpec.DesiredState, nil
	}

	// Use nmstatectl policy validation for nmstate backend
	nmstatePolicy := struct {
		Capture      map[string]string `json:"capture,omitempty"`
		DesiredState nmstateapi.State  `json:"desiredState,omitempty"`
	}{
		Capture:      policySpec.Capture,
		DesiredState: policySpec.DesiredState,
	}

	nmstatePolicyRaw, err := json.Marshal(nmstatePolicy)
	if err != nil {
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
			nmstateapi.State{}, err
	}
	cachedStateRaw := []byte{}
	if len(cachedState) > 0 {
		cachedStateRaw, err = json.Marshal(cachedState)
		if err != nil {
			return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
				nmstateapi.State{}, err
		}
	}

	output, capturedStateRaw, err := nmstatectl.Policy(nmstatePolicyRaw, []byte(currentState.Raw), cachedStateRaw)
	if err != nil {
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
			nmstateapi.State{}, err
	}

	capturedState := map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{}
	if err := json.Unmarshal(capturedStateRaw, &capturedState); err != nil {
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{},
			nmstateapi.State{}, err
	}

	return capturedState, nmstateapi.State{Raw: output}, nil
}
