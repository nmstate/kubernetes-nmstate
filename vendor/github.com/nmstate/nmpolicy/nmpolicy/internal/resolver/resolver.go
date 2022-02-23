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
	"errors"
	"fmt"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/expression"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/types"
)

type Resolver struct{}

type resolver struct {
	currentState       types.NMState
	capturedStates     types.CapturedStates
	captureExpressions types.CaptureExpressions
	captureASTPool     types.CaptureASTPool
	currentNode        *ast.Node
	currentExpression  *string
}

func New() Resolver {
	return Resolver{}
}

func newResolver() *resolver {
	return &resolver{
		currentState:   types.NMState{},
		capturedStates: types.CapturedStates{},
		captureASTPool: types.CaptureASTPool{},
	}
}

func (Resolver) Resolve(captureExpressions types.CaptureExpressions,
	captureASTPool types.CaptureASTPool,
	currentState types.NMState,
	capturedStates types.CapturedStates) (types.CapturedStates, error) {
	r := newResolver()
	r.currentState = currentState
	r.captureASTPool = captureASTPool
	r.captureExpressions = captureExpressions
	if capturedStates != nil {
		r.capturedStates = capturedStates
	}
	capturedStates, err := r.resolve()
	return capturedStates, r.wrapErrorWithCurrentExpression(err)
}

func (Resolver) ResolveCaptureEntryPath(expr string, captureEntryPathAST ast.Node,
	capturedStates types.CapturedStates) (interface{}, error) {
	r := newResolver()
	r.currentExpression = &expr
	r.capturedStates = capturedStates
	r.currentNode = &captureEntryPathAST
	resolvedCaptureEntryPath, err := r.resolveCaptureEntryPath()
	return resolvedCaptureEntryPath, r.wrapErrorWithCurrentExpression(err)
}

func (r *resolver) resolve() (types.CapturedStates, error) {
	for captureEntryName := range r.captureASTPool {
		if _, err := r.resolveCaptureEntryName(captureEntryName); err != nil {
			return nil, wrapWithResolveError(err)
		}
	}
	return r.capturedStates, nil
}

func (r *resolver) resolveCaptureEntryName(captureEntryName string) (types.NMState, error) {
	expr, ok := r.captureExpressions[captureEntryName]
	if ok {
		r.currentExpression = &expr
	}
	capturedStateEntry, ok := r.capturedStates[captureEntryName]
	if ok {
		return capturedStateEntry.State, nil
	}
	captureASTEntry, ok := r.captureASTPool[captureEntryName]
	if !ok {
		return nil, fmt.Errorf("capture entry '%s' not found", captureEntryName)
	}
	r.currentNode = &captureASTEntry
	var err error
	capturedStateEntry = types.CapturedState{}
	capturedStateEntry.State, err = r.resolveCaptureASTEntry()
	if err != nil {
		return nil, err
	}
	r.capturedStates[captureEntryName] = capturedStateEntry
	return capturedStateEntry.State, nil
}

func (r *resolver) resolveCaptureASTEntry() (types.NMState, error) {
	if r.currentNode.EqFilter != nil {
		return r.resolveEqFilter()
	} else if r.currentNode.Replace != nil {
		return r.resolveReplace()
	}
	return nil, fmt.Errorf("root node has unsupported operation : %s", *r.currentNode)
}

func (r *resolver) resolveEqFilter() (types.NMState, error) {
	operator := r.currentNode.EqFilter
	filteredState, err := r.resolveTernaryOperator(operator, filter)
	if err != nil {
		return nil, wrapWithEqFilterError(err)
	}
	return filteredState, nil
}

func (r *resolver) resolveReplace() (types.NMState, error) {
	operator := r.currentNode.Replace
	replacedState, err := r.resolveTernaryOperator(operator, replace)
	if err != nil {
		return nil, wrapWithResolveError(err)
	}
	return replacedState, nil
}

func (r *resolver) resolveTernaryOperator(operator *ast.TernaryOperator,
	resolverFunc func(map[string]interface{}, ast.VariadicOperator, interface{}) (map[string]interface{}, error)) (types.NMState, error) {
	operatorNode := r.currentNode
	r.currentNode = &(*operator)[0]
	inputSource, err := r.resolveInputSource()
	if err != nil {
		return nil, err
	}

	r.currentNode = &(*operator)[1]
	path, err := r.resolvePath()
	if err != nil {
		return nil, err
	}
	r.currentNode = &(*operator)[2]
	value, err := r.resolveStringOrCaptureEntryPath()
	if err != nil {
		return nil, err
	}

	r.currentNode = operatorNode
	resolvedState, err := resolverFunc(inputSource, path.steps, value)
	if err != nil {
		return nil, err
	}
	return resolvedState, nil
}

func (r *resolver) resolveInputSource() (types.NMState, error) {
	if ast.CurrentStateIdentity().DeepEqual(r.currentNode.Terminal) {
		return r.currentState, nil
	} else if r.currentNode.Path != nil {
		resolvedPath, err := r.resolvePath()
		if err != nil {
			return nil, err
		}
		if resolvedPath.captureEntryName == "" {
			return nil, fmt.Errorf("invalid path input source (%s), only capture reference is supported", r.currentNode)
		}
		capturedState, err := r.resolveCaptureEntryName(resolvedPath.captureEntryName)
		if err != nil {
			return nil, err
		}
		return capturedState, nil
	}

	return nil, fmt.Errorf("invalid input source (%s), only current state or capture reference is supported", *r.currentNode)
}

func (r *resolver) resolveStringOrCaptureEntryPath() (interface{}, error) {
	if r.currentNode.Str != nil {
		return *r.currentNode.Str, nil
	} else if r.currentNode.Path != nil {
		return r.resolveCaptureEntryPath()
	} else {
		return nil, fmt.Errorf("not supported value. Only string or capture entry path are supported")
	}
}

func (r *resolver) resolveCaptureEntryPath() (interface{}, error) {
	resolvedPath, err := r.resolvePath()
	if err != nil {
		return nil, err
	}
	if resolvedPath.captureEntryName == "" {
		return nil, fmt.Errorf("not supported filtered value path. Only paths with a capture entry reference are supported")
	}
	capturedStateEntry, err := r.resolveCaptureEntryName(resolvedPath.captureEntryName)
	if err != nil {
		return nil, err
	}
	return resolvedPath.walkState(capturedStateEntry)
}

func (r *resolver) resolvePath() (*captureEntryNameAndSteps, error) {
	if r.currentNode.Path == nil {
		return nil, fmt.Errorf("invalid path type %T", *r.currentNode)
	} else if len(*r.currentNode.Path) == 0 {
		return nil, fmt.Errorf("empty path length")
	} else if (*r.currentNode.Path)[0].Identity == nil {
		return nil, fmt.Errorf("path first step has to be an identity")
	}
	resolvedPath := captureEntryNameAndSteps{
		steps: *r.currentNode.Path,
	}
	if *resolvedPath.steps[0].Identity == "capture" {
		const captureRefSize = 2
		if len(resolvedPath.steps) < captureRefSize || resolvedPath.steps[1].Identity == nil {
			return nil, fmt.Errorf("path capture ref is missing capture entry name")
		}
		resolvedPath.captureEntryName = *resolvedPath.steps[1].Identity
		if len(resolvedPath.steps) > captureRefSize {
			resolvedPath.steps = resolvedPath.steps[2:len(resolvedPath.steps)]
		}
	}
	return &resolvedPath, nil
}

func (r resolver) wrapErrorWithCurrentExpression(err error) error {
	if err == nil {
		return nil
	}

	var pathError PathError
	errorNode := r.currentNode
	if errors.As(err, &pathError) {
		errorNode = pathError.errorNode
	}

	if errorNode == nil || r.currentExpression == nil {
		return err
	}
	return expression.WrapError(err, *r.currentExpression, errorNode.Meta.Position)
}
