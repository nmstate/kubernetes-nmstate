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
	"reflect"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
)

func filter(
	inputState map[string]interface{},
	pathSteps ast.VariadicOperator,
	operator func(interface{}, interface{}) bool,
	expectedValue interface{}) (map[string]interface{}, error) {
	filtered, err := visitState(newPath(pathSteps), inputState, &filterVisitor{
		operator:      operator,
		expectedValue: expectedValue,
	})

	if err != nil {
		return nil, fmt.Errorf("failed applying operation on the path: %w", err)
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
func eqfilter(
	inputState map[string]interface{},
	pathSteps ast.VariadicOperator,
	expectedValue interface{}) (map[string]interface{}, error) {
	return filter(inputState, pathSteps, func(lhs, rhs interface{}) bool { return lhs == rhs }, expectedValue)
}
func nefilter(
	inputState map[string]interface{},
	pathSteps ast.VariadicOperator,
	expectedValue interface{}) (map[string]interface{}, error) {
	return filter(inputState, pathSteps, func(lhs, rhs interface{}) bool { return lhs != rhs }, expectedValue)
}

type filterVisitor struct {
	mergeVisitResult bool
	operator         func(interface{}, interface{}) bool
	expectedValue    interface{}
}

func (e filterVisitor) visitLastMap(p path, mapToFilter map[string]interface{}) (interface{}, error) {
	obtainedValue, ok := mapToFilter[*p.currentStep.Identity]
	if !ok {
		return nil, nil
	}

	// Filter by the path since there is no value to compare
	if e.expectedValue == nil {
		return map[string]interface{}{*p.currentStep.Identity: obtainedValue}, nil
	}

	if reflect.TypeOf(obtainedValue) != reflect.TypeOf(e.expectedValue) {
		return nil, pathError(p.currentStep, `type missmatch: the value in the path doesn't match the value to filter. `+
			`"%T" != "%T" -> %+v != %+v`, obtainedValue, e.expectedValue, obtainedValue, e.expectedValue)
	}
	if e.operator(obtainedValue, e.expectedValue) {
		return mapToFilter, nil
	}
	return nil, nil
}

func (e filterVisitor) visitLastSlice(p path, sliceToVisit []interface{}) (interface{}, error) {
	if p.currentStep.Identity != nil {
		return e.visitSlice(p, sliceToVisit)
	}
	return nil, pathError(p.currentStep, "filtering lists index with equal not implemented")
}

func (e filterVisitor) visitMap(p path, mapToVisit map[string]interface{}) (interface{}, error) {
	if p.currentStep.Number != nil {
		return nil, pathError(p.currentStep, "failed filtering map: path with index not supported")
	}
	interfaceToVisit, ok := mapToVisit[*p.currentStep.Identity]
	if !ok {
		return nil, nil
	}
	visitResult, err := visitState(p.nextStep(), interfaceToVisit, &e)
	if err != nil {
		return nil, err
	}
	if visitResult == nil {
		return nil, nil
	}
	filteredMap := map[string]interface{}{}
	if e.mergeVisitResult {
		for k, v := range mapToVisit {
			filteredMap[k] = v
		}
	}
	filteredMap[*p.currentStep.Identity] = visitResult
	return filteredMap, nil
}

func (e filterVisitor) visitSlice(p path, sliceToVisit []interface{}) (interface{}, error) {
	if p.currentStep.Number != nil {
		return nil, pathError(p.currentStep, "failed filtering slice: path with index not supported")
	}

	filteredSlice := []interface{}{}
	hasVisitResult := false
	for _, interfaceToVisit := range sliceToVisit {
		// Filter only the first slice by forcing "mergeVisitResult" to true
		// for the the following ones.
		visitResult, err := visitState(p, interfaceToVisit, &filterVisitor{
			mergeVisitResult: true,
			operator:         e.operator,
			expectedValue:    e.expectedValue})
		if err != nil {
			return nil, err
		}
		if visitResult != nil {
			hasVisitResult = true
			filteredSlice = append(filteredSlice, visitResult)
		} else if e.mergeVisitResult {
			filteredSlice = append(filteredSlice, interfaceToVisit)
		}
	}

	if !hasVisitResult {
		return nil, nil
	}
	return filteredSlice, nil
}
