/*
 * Copyright 2022 NMPolicy Authors.
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

func walk(inputState map[string]interface{}, pathSteps ast.VariadicOperator) (interface{}, error) {
	visitResult, err := visitState(newPath(pathSteps), inputState, &walkOpVisitor{})
	if err != nil {
		return nil, fmt.Errorf("failed walking path: %w", err)
	}

	return visitResult, nil
}

type walkOpVisitor struct{}

func (walkOpVisitor) visitLastMap(p path, mapToAccess map[string]interface{}) (interface{}, error) {
	return accessMapWithCurrentStep(p, mapToAccess)
}

func (walkOpVisitor) visitLastSlice(p path, sliceToAccess []interface{}) (interface{}, error) {
	return accessSliceWithCurrentStep(p, sliceToAccess)
}

func (w walkOpVisitor) visitSlice(p path, sliceToVisit []interface{}) (interface{}, error) {
	interfaceToVisit, err := accessSliceWithCurrentStep(p, sliceToVisit)
	if err != nil {
		return nil, err
	}
	return visitState(p.nextStep(), interfaceToVisit, &w)
}

func (w walkOpVisitor) visitMap(p path, mapToVisit map[string]interface{}) (interface{}, error) {
	interfaceToVisit, err := accessMapWithCurrentStep(p, mapToVisit)
	if err != nil {
		return nil, err
	}
	return visitState(p.nextStep(), interfaceToVisit, &w)
}

func accessMapWithCurrentStep(p path, mapToAccess map[string]interface{}) (interface{}, error) {
	if p.currentStep.Identity == nil {
		return nil, pathError(p.currentStep, "unexpected non identity step for smap state '%+v'", mapToAccess)
	}
	v, ok := mapToAccess[*p.currentStep.Identity]
	if !ok {
		return nil, pathError(p.currentStep, "step not found at map state '%+v'", mapToAccess)
	}
	return v, nil
}

func accessSliceWithCurrentStep(p path, sliceToAccess []interface{}) (interface{}, error) {
	if p.currentStep.Number == nil {
		return nil, pathError(p.currentStep, "unexpected non numeric step for slice state '%+v'", sliceToAccess)
	}
	if len(sliceToAccess) <= *p.currentStep.Number {
		return nil, pathError(p.currentStep, "step not found at slice state '%+v'", sliceToAccess)
	}
	return sliceToAccess[*p.currentStep.Number], nil
}
