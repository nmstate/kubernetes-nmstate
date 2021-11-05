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

package capture

import (
	"fmt"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
	"github.com/nmstate/nmpolicy/nmpolicy/types"
)

type Capture struct {
	lexer    Lexer
	parser   Parser
	resolver Resolver
}

type Lexer interface {
	Lex(expression string) ([]lexer.Token, error)
}

type Parser interface {
	Parse([]lexer.Token) (ast.Node, error)
}

type Resolver interface {
	Resolve(astPool map[string]ast.Node, state []byte) (map[string]types.CaptureState, error)
}

func New(leXer Lexer, parser Parser, resolver Resolver) Capture {
	return Capture{
		lexer:    leXer,
		parser:   parser,
		resolver: resolver,
	}
}

func (c Capture) Resolve(
	capturesExpr map[string]string,
	capturesCache map[string]types.CaptureState,
	state []byte) (map[string]types.CaptureState, error) {
	if len(capturesExpr) == 0 || len(state) == 0 && len(capturesCache) == 0 {
		return nil, nil
	}

	capturesState := filterCacheBasedOnExprCaptures(capturesCache, capturesExpr)
	capturesExpr = filterOutExprBasedOnCachedCaptures(capturesExpr, capturesCache)

	astPool := map[string]ast.Node{}
	for capID, capExpr := range capturesExpr {
		tokens, err := c.lexer.Lex(capExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
		}

		astRoot, err := c.parser.Parse(tokens)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
		}

		astPool[capID] = astRoot
	}

	resolvedCapsState, err := c.resolver.Resolve(astPool, state)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
	}

	for capID, capExpr := range resolvedCapsState {
		capturesState[capID] = capExpr
	}

	return capturesState, nil
}

func filterOutExprBasedOnCachedCaptures(capturesExpr map[string]string, capturesCache map[string]types.CaptureState) map[string]string {
	for capID := range capturesCache {
		delete(capturesExpr, capID)
	}
	return capturesExpr
}

func filterCacheBasedOnExprCaptures(capsState map[string]types.CaptureState, capsExpr map[string]string) map[string]types.CaptureState {
	caps := map[string]types.CaptureState{}

	for capID := range capsExpr {
		if capState, ok := capsState[capID]; ok {
			state := append([]byte{}, capState.State...)

			caps[capID] = types.CaptureState{
				State:    state,
				MetaInfo: capState.MetaInfo,
			}
		}
	}
	return caps
}
