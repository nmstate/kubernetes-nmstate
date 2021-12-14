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

package resolver

import (
	"fmt"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
)

func filter(inputState map[string]interface{}, path ast.VariadicOperator, expectedNode ast.Node) (map[string]interface{}, error) {
	filtered, err := applyFuncOnMap(path, inputState, expectedNode, mapContainsValue, true, true)

	if err != nil {
		return nil, fmt.Errorf("failed applying operation on the path: %v", err)
	}

	if filtered == nil {
		return nil, nil
	}

	filteredMap, ok := filtered.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed converting filtering result to a map")
	}
	return filteredMap, nil
}

func isEqual(obtainedValue interface{}, desiredValue ast.Node) (bool, error) {
	if desiredValue.String != nil {
		stringToCompare, ok := obtainedValue.(string)
		if !ok {
			return false, fmt.Errorf("the value %v of type %T not supported,"+
				"curretly only string values are supported", obtainedValue, obtainedValue)
		}
		return stringToCompare == *desiredValue.String, nil
	}

	return false, fmt.Errorf("the desired value %v is not supported. Curretly only string values are supported", desiredValue)
}

func mapContainsValue(mapToFilter map[string]interface{}, filterKey string, expectedNode ast.Node) (interface{}, error) {
	obtainedValue, ok := mapToFilter[filterKey]
	if !ok {
		return nil, fmt.Errorf("cannot find key %s in %v", filterKey, mapToFilter)
	}
	valueIsEqual, err := isEqual(obtainedValue, expectedNode)
	if err != nil {
		return nil, fmt.Errorf("error comparing the expected and obtained values : %v", err)
	}
	if valueIsEqual {
		return mapToFilter, nil
	}
	return nil, nil
}
