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

	"sigs.k8s.io/yaml"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	"github.com/nmstate/nmpolicy/nmpolicy/types"
)

type Resolver struct{}

func New() Resolver {
	return Resolver{}
}

func (r Resolver) Resolve(astPool map[string]ast.Node, state []byte) (map[string]types.CaptureState, error) {
	currentState := make(map[string]interface{})
	err := yaml.Unmarshal(state, &currentState)
	if err != nil {
		return nil, wrapWithResolveError(err)
	}
	resolvedASTs := make(map[string]types.CaptureState)
	for captureName, ast := range astPool {
		resolvedAST, err := r.resolveAST(ast, currentState)
		if err != nil {
			return nil, wrapWithResolveError(err)
		}
		rawAST, err := yaml.Marshal(resolvedAST)
		if err != nil {
			return nil, wrapWithResolveError(err)
		}
		resolvedASTs[captureName] = types.CaptureState{State: rawAST, MetaInfo: types.MetaInfo{}}
	}
	return resolvedASTs, nil
}

func (r Resolver) resolveAST(captureAST ast.Node, currentState map[string]interface{}) (map[string]interface{}, error) {
	if captureAST.EqFilter != nil {
		inputSource, err := r.resolveInputSource(captureAST.EqFilter[0], currentState)
		if err != nil {
			return nil, err
		}

		path, err := r.resolvePath(captureAST.EqFilter[1])
		if err != nil {
			return nil, err
		}
		filteredValue, err := r.resolveFilteredValue(captureAST.EqFilter[2])
		if err != nil {
			return nil, err
		}

		return filter(inputSource, *path, *filteredValue)
	}
	return nil, fmt.Errorf("root node has unsupported operation : %v", captureAST)
}

func (r Resolver) resolveInputSource(inputSourceNode ast.Node, currentState map[string]interface{}) (map[string]interface{}, error) {
	if ast.CurrentStateIdentity().DeepEqual(inputSourceNode.Terminal) {
		return currentState, nil
	}

	return nil, fmt.Errorf("not supported input source %v. Only the current state is supported", inputSourceNode)
}

func (r Resolver) resolvePath(pathNode ast.Node) (*ast.VariadicOperator, error) {
	if pathNode.Path == nil {
		return nil, fmt.Errorf("invalid path type %T", pathNode)
	}

	return pathNode.Path, nil
}

func (r Resolver) resolveFilteredValue(filteredValueNode ast.Node) (*ast.Node, error) {
	if filteredValueNode.String == nil {
		return nil, fmt.Errorf("not supported filtered value %v. Only string is supported", filteredValueNode)
	}
	return &filteredValueNode, nil
}
