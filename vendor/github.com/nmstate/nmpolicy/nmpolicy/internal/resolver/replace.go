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
	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
)

func replace(inputState map[string]interface{}, pathSteps ast.VariadicOperator, replaceValue interface{}) (map[string]interface{}, error) {
	replaced, err := visitState(newPath(pathSteps), inputState, &replaceOpVisitor{replaceValue})

	if err != nil {
		return nil, replaceError("failed applying operation on the path: %w", err)
	}

	replacedMap, ok := replaced.(map[string]interface{})
	if !ok {
		return nil, replaceError("failed converting result to a map")
	}
	return replacedMap, nil
}

type replaceOpVisitor struct {
	replaceValue interface{}
}

func (r replaceOpVisitor) visitLastMap(p path, inputMap map[string]interface{}) (interface{}, error) {
	modifiedMap := map[string]interface{}{}
	for k, v := range inputMap {
		modifiedMap[k] = v
	}

	modifiedMap[*p.currentStep.Identity] = r.replaceValue
	return modifiedMap, nil
}

func (r replaceOpVisitor) visitLastSlice(p path, sliceToVisit []interface{}) (interface{}, error) {
	if p.currentStep.Identity != nil {
		return r.visitSlice(p, sliceToVisit)
	}
	return nil, pathError(p.currentStep, "replacing lists value at index not implemented")
}

func (r replaceOpVisitor) visitMap(p path, mapToVisit map[string]interface{}) (interface{}, error) {
	if p.currentStep.Number != nil {
		return nil, pathError(p.currentStep, "failed replacing map: path with index not supported")
	}
	interfaceToVisit, ok := mapToVisit[*p.currentStep.Identity]
	if !ok {
		interfaceToVisit = map[string]interface{}{}
	}

	visitResult, err := visitState(p.nextStep(), interfaceToVisit, &r)
	if err != nil {
		return nil, err
	}

	replacedMap := map[string]interface{}{}
	for k, v := range mapToVisit {
		replacedMap[k] = v
	}
	replacedMap[*p.currentStep.Identity] = visitResult
	return replacedMap, nil
}

func (r replaceOpVisitor) visitSlice(p path, sliceToVisit []interface{}) (interface{}, error) {
	if p.currentStep.Number != nil {
		return nil, pathError(p.currentStep, "failed replacing slice: path with index not supported")
	}

	replacedSlice := make([]interface{}, len(sliceToVisit))
	for i, interfaceToVisit := range sliceToVisit {
		visitResult, err := visitState(p, interfaceToVisit, &r)
		if err != nil {
			return nil, err
		}
		replacedSlice[i] = visitResult
	}
	return replacedSlice, nil
}
