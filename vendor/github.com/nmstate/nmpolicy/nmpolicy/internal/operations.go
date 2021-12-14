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

package internal

import (
	"fmt"
	"time"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/capture"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/expander"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/parser"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/resolver"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/types"
	nmpolicytypes "github.com/nmstate/nmpolicy/nmpolicy/types"
)

func GenerateState(policySpec types.PolicySpec, currentState types.NMState, cachedState types.CachedState) (types.GeneratedState, error) {
	var (
		capturedStates types.CapturedStates
		desiredState   types.NMState
	)

	if policySpec.DesiredState != nil {
		capResolver := capture.New(lexer.New(), parser.New(), resolver.New())
		var err error
		capturedStates, err = capResolver.Resolve(policySpec.Capture, cachedState.CapturedStates, currentState)
		if err != nil {
			return types.GeneratedState{}, fmt.Errorf("failed to generate state, err: %v", err)
		}

		captureEntryPathResolver, err := capture.NewCaptureEntry(capturedStates)
		if err != nil {
			return types.GeneratedState{}, fmt.Errorf("failed to generate state, err: %v", err)
		}

		stateExpander := expander.New(captureEntryPathResolver)
		desiredState, err = stateExpander.Expand(policySpec.DesiredState)
		if err != nil {
			return types.GeneratedState{}, fmt.Errorf("failed to generate state, err: %v", err)
		}
	}

	timestamp := time.Now().UTC()
	timestampCapturesState(capturedStates, timestamp)
	return types.GeneratedState{
		Cache:        types.CachedState{CapturedStates: capturedStates},
		DesiredState: desiredState,
		MetaInfo: nmpolicytypes.MetaInfo{
			Version:   "0",
			TimeStamp: timestamp,
		},
	}, nil
}

func timestampCapturesState(capturedStates types.CapturedStates, timeStamp time.Time) {
	for captureID, capturedState := range capturedStates {
		if capturedState.MetaInfo.TimeStamp.IsZero() {
			capturedState.MetaInfo.TimeStamp = timeStamp
			capturedStates[captureID] = capturedState
		}
	}
}
