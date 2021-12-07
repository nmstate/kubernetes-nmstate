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

package types

import (
	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	nmpolicytypes "github.com/nmstate/nmpolicy/nmpolicy/types"
)

type NMState map[string]interface{}
type CaptureExpressions map[string]string
type CapturedStates map[string]CapturedState
type CaptureASTPool map[string]ast.Node

type PolicySpec struct {
	Capture      CaptureExpressions
	DesiredState NMState
}

type CachedState struct {
	CapturedStates CapturedStates
}

type GeneratedState struct {
	Cache        CachedState
	DesiredState NMState
	MetaInfo     nmpolicytypes.MetaInfo
}

type CapturedState struct {
	State    NMState
	MetaInfo nmpolicytypes.MetaInfo
}
