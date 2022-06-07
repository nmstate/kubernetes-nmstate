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

type stateVisitor interface {
	visitLastMap(path, map[string]interface{}) (interface{}, error)
	visitLastSlice(path, []interface{}) (interface{}, error)
	visitMap(path, map[string]interface{}) (interface{}, error)
	visitSlice(path, []interface{}) (interface{}, error)
}

func visitState(p path, inputState interface{}, v stateVisitor) (interface{}, error) {
	originalMap, isMap := inputState.(map[string]interface{})
	if isMap {
		if p.hasMoreSteps() {
			if p.currentStep.Identity == nil {
				return nil, pathError(p.currentStep, "unexpected non identity step for map state '%+v'", originalMap)
			}
			return v.visitMap(p, originalMap)
		}
		return v.visitLastMap(p, originalMap)
	}

	originalSlice, isSlice := inputState.([]interface{})
	if isSlice {
		if p.hasMoreSteps() {
			return v.visitSlice(p, originalSlice)
		}
		return v.visitLastSlice(p, originalSlice)
	}
	return nil, pathError(p.currentStep, "invalid type %T for identity step '%v'", inputState, *p.currentStep)
}
