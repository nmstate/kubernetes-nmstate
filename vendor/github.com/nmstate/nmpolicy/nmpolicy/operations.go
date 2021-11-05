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
	"time"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/capture"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/parser"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/resolver"
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
func GenerateState(nmpolicy types.PolicySpec, currentState []byte, cache types.CachedState) (types.GeneratedState, error) {
	var capturesState map[string]types.CaptureState
	var desiredState []byte

	if nmpolicy.DesiredState != nil {
		desiredState = append(desiredState, nmpolicy.DesiredState...)

		capResolver := capture.New(lexer.New(), parser.New(), resolver.New())
		var err error
		capturesState, err = capResolver.Resolve(nmpolicy.Capture, cache.Capture, currentState)
		if err != nil {
			return types.GeneratedState{}, fmt.Errorf("failed to generate state, err: %v", err)
		}

		// TODO: Resolve/expend the desired state.
	}

	timestamp := time.Now().UTC()
	timestampCapturesState(capturesState, timestamp)
	return types.GeneratedState{
		Cache:        types.CachedState{Capture: capturesState},
		DesiredState: desiredState,
		MetaInfo: types.MetaInfo{
			Version:   "0",
			TimeStamp: timestamp,
		},
	}, nil
}

func timestampCapturesState(capturesState map[string]types.CaptureState, timeStamp time.Time) {
	for captureID, captureState := range capturesState {
		if captureState.MetaInfo.TimeStamp.IsZero() {
			captureState.MetaInfo.TimeStamp = timeStamp
			capturesState[captureID] = captureState
		}
	}
}
