/*
 * Copyright 2021 NMPolicy Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nmpolicy

import (
	"fmt"

	yaml "sigs.k8s.io/yaml"

	"github.com/nmstate/nmpolicy/nmpolicy/internal"
	internaltypes "github.com/nmstate/nmpolicy/nmpolicy/internal/types"
	"github.com/nmstate/nmpolicy/nmpolicy/types"
)

// GenerateState creates a NMPolicy state based on the given input data:
// - NMPolicy spec.
// - NMState state, representing a given current state.
// - Cache state which includes (already resolved) named captures.
//
// GenerateState returns a generated state which includes:
// - Desired State: The NMState state which has been built by the policy.
// - Cache: Named NMState states which have been resolved by the policy.
//          Can be saved for use as cache data (passed as input).
// - Meta Info: Extended information about the generated state (e.g. the policy version).
//
// On failure, an error is returned.
func GenerateState(policySpec types.PolicySpec,
	currentState []byte, cachedState types.CachedState) (generatedState types.GeneratedState, err error) {
	internalPolicySpec, err := toInternalPolicySpec(policySpec)
	if err != nil {
		return generatedState, fmt.Errorf("failed converting to internal policy spec: %v", err)
	}
	internalCurrentState, err := toInternalNMState(currentState)
	if err != nil {
		return generatedState, fmt.Errorf("failed converting to internal current state: %v", err)
	}
	internalCachedState, err := toInternalCachedState(cachedState)
	if err != nil {
		return generatedState, fmt.Errorf("failed converting to internal cached state: %v", err)
	}

	internalGeneratedState, err := internal.GenerateState(internalPolicySpec, internalCurrentState, internalCachedState)
	if err != nil {
		return generatedState, err
	}

	generatedState, err = toGeneratedState(internalGeneratedState)
	if err != nil {
		return generatedState, err
	}
	return generatedState, nil
}

func toInternalPolicySpec(policySpec types.PolicySpec) (internaltypes.PolicySpec, error) {
	internalDesiredState, err := toInternalNMState(policySpec.DesiredState)
	if err != nil {
		return internaltypes.PolicySpec{}, err
	}
	return internaltypes.PolicySpec{
		DesiredState: internalDesiredState,
		Capture:      internaltypes.CaptureExpressions(policySpec.Capture),
	}, nil
}

func toInternalNMState(nmState []byte) (internaltypes.NMState, error) {
	internalNMState := internaltypes.NMState{}
	if err := yaml.Unmarshal(nmState, &internalNMState); err != nil {
		return nil, fmt.Errorf("failed converting to internal NMState: %w", err)
	}
	return internalNMState, nil
}

func toNMState(internalNMState internaltypes.NMState) ([]byte, error) {
	if len(internalNMState) == 0 {
		return nil, nil
	}
	nmState, err := yaml.Marshal(internalNMState)
	if err != nil {
		return nil, fmt.Errorf("failed converting from internal NMState: %w", err)
	}
	return nmState, nil
}

func toInternalCapturedState(capturedState types.CaptureState) (internaltypes.CapturedState, error) {
	internalState, err := toInternalNMState(capturedState.State)
	if err != nil {
		return internaltypes.CapturedState{}, err
	}
	return internaltypes.CapturedState{
		State:    internalState,
		MetaInfo: capturedState.MetaInfo,
	}, nil
}

func toCapturedState(internalCapturedState internaltypes.CapturedState) (types.CaptureState, error) {
	state, err := toNMState(internalCapturedState.State)
	if err != nil {
		return types.CaptureState{}, err
	}
	return types.CaptureState{
		State:    state,
		MetaInfo: internalCapturedState.MetaInfo,
	}, nil
}

func toInternalCachedState(cachedState types.CachedState) (internaltypes.CachedState, error) {
	internalCachedState := internaltypes.CachedState{
		CapturedStates: internaltypes.CapturedStates{},
	}
	for captureEntryName, capturedState := range cachedState.Capture {
		internalCapturedState, err := toInternalCapturedState(capturedState)
		if err != nil {
			return internaltypes.CachedState{}, err
		}
		internalCachedState.CapturedStates[captureEntryName] = internalCapturedState
	}
	return internalCachedState, nil
}

func toCachedState(internalCachedState internaltypes.CachedState) (types.CachedState, error) {
	if len(internalCachedState.CapturedStates) == 0 {
		return types.CachedState{}, nil
	}
	cachedState := types.CachedState{
		Capture: map[string]types.CaptureState{},
	}
	for captureEntryName, internalCapturedState := range internalCachedState.CapturedStates {
		capturedState, err := toCapturedState(internalCapturedState)
		if err != nil {
			return types.CachedState{}, err
		}
		cachedState.Capture[captureEntryName] = capturedState
	}
	return cachedState, nil
}

func toGeneratedState(internalGeneratedState internaltypes.GeneratedState) (types.GeneratedState, error) {
	desiredState, err := toNMState(internalGeneratedState.DesiredState)
	if err != nil {
		return types.GeneratedState{}, err
	}
	cachedState, err := toCachedState(internalGeneratedState.Cache)
	if err != nil {
		return types.GeneratedState{}, err
	}
	return types.GeneratedState{
		DesiredState: desiredState,
		Cache:        cachedState,
	}, nil
}
