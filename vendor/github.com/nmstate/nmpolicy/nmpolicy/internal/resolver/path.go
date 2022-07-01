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

type path struct {
	steps            []ast.Node
	currentStepIndex int
	currentStep      *ast.Node
}

func newPath(steps []ast.Node) path {
	return path{
		steps:            steps,
		currentStepIndex: 0,
		currentStep:      &steps[0],
	}
}

func (p path) nextStep() path {
	if p.hasMoreSteps() {
		p.currentStepIndex++
	}
	p.currentStep = &p.steps[p.currentStepIndex]
	return p
}

func (p path) hasMoreSteps() bool {
	return p.currentStepIndex+1 < len(p.steps)
}
