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

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nmstate/nmpolicy/nmpolicy"
	nmpolicytypes "github.com/nmstate/nmpolicy/nmpolicy/types"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

var (
	log = logf.Log.WithName("policy")
)

type NMPolicyGenerator interface {
	GenerateState(
		nmpolicySpec nmpolicytypes.PolicySpec,
		currentState []byte,
		cache nmpolicytypes.CachedState,
	) (nmpolicytypes.GeneratedState, error)
}

type GenerateStateWithNMPolicy struct{}

func (GenerateStateWithNMPolicy) GenerateState(
	nmpolicySpec nmpolicytypes.PolicySpec,
	currentState []byte,
	cache nmpolicytypes.CachedState,
) (nmpolicytypes.GeneratedState, error) {
	return nmpolicy.GenerateState(nmpolicySpec, currentState, cache)
}

// The method generates the state using the default NMPolicyGenerator
func GenerateState(desiredState nmstateapi.State,
	policySpec nmstateapi.NodeNetworkConfigurationPolicySpec,
	currentState nmstateapi.State,
	cachedState map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState) (
	map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState, /* resolved captures */
	nmstateapi.State, /* updated desired state */
	error) {
	return GenerateStateWithStateGenerator(GenerateStateWithNMPolicy{}, desiredState, policySpec, currentState, cachedState)
}

// The method generates the state using NMPolicyGenerator.GenerateState and then converts the returned value to the match the enactment api
func GenerateStateWithStateGenerator(stateGenerator NMPolicyGenerator,
	desiredState nmstateapi.State,
	policySpec nmstateapi.NodeNetworkConfigurationPolicySpec,
	currentState nmstateapi.State,
	cachedState map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState) (
	map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState,
	nmstateapi.State, error) {

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

	return capturedState, nmstateapi.State{Raw: []byte(output)}, nil
}
