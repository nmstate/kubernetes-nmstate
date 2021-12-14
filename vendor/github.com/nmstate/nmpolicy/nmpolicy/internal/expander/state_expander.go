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

package expander

import (
	"fmt"
	"regexp"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/types"
)

type StateExpander struct {
	capResolver CapturePathResolver
}

type CapturePathResolver interface {
	ResolveCaptureEntryPath(capturePath string) (interface{}, error)
}

func New(capResolver CapturePathResolver) StateExpander {
	return StateExpander{capResolver: capResolver}
}

func (c StateExpander) Expand(desiredState types.NMState) (types.NMState, error) {
	expandedState, err := c.expandState(map[string]interface{}(desiredState))
	if err != nil {
		return nil, fmt.Errorf("failed expanding desired state: %v", err)
	}
	return types.NMState(expandedState.(map[string]interface{})), nil
}

func (c StateExpander) expandState(state interface{}) (interface{}, error) {
	switch stateValue := state.(type) {
	case nil:
		return state, nil
	case string:
		return c.expandString(stateValue)
	case map[string]interface{}:
		return c.expandMap(stateValue)
	case []interface{}:
		return c.exapndSlice(stateValue)
	default:
		return state, nil
	}
}

func (c StateExpander) exapndSlice(sliceState []interface{}) ([]interface{}, error) {
	for i, value := range sliceState {
		expandedValue, err := c.expandState(value)
		if err != nil {
			return nil, err
		}
		sliceState[i] = expandedValue
	}
	return sliceState, nil
}

func (c StateExpander) expandMap(mapState map[string]interface{}) (map[string]interface{}, error) {
	for key, value := range mapState {
		expandedValue, err := c.expandState(value)
		mapState[key] = expandedValue
		if err != nil {
			return nil, err
		}
	}
	return mapState, nil
}

func (c StateExpander) expandString(stringState string) (interface{}, error) {
	re := regexp.MustCompile(`^{{ (.*) }}$`)
	submatch := re.FindStringSubmatch(stringState)

	if len(submatch) == 0 {
		return stringState, nil
	}

	const captureSubmatchLength = 2
	if len(submatch) != captureSubmatchLength {
		return nil, fmt.Errorf("the capture expression has wrong format %s", stringState)
	}

	capturePath := submatch[1]
	resolvedPath, err := c.capResolver.ResolveCaptureEntryPath(capturePath)

	if err != nil {
		return nil, err
	}

	return resolvedPath, nil
}
