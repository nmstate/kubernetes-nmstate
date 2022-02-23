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
	"github.com/nmstate/nmpolicy/nmpolicy/internal/types"
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
	Parse(string, []lexer.Token) (ast.Node, error)
}

type Resolver interface {
	Resolve(captureExpressions types.CaptureExpressions, captureASTPool types.CaptureASTPool,
		state types.NMState, capturedStates types.CapturedStates) (types.CapturedStates, error)
	ResolveCaptureEntryPath(expression string, captureEntryPathAST ast.Node, capturedStates types.CapturedStates) (interface{}, error)
}

func New(leXer Lexer, parser Parser, resolver Resolver) Capture {
	return Capture{
		lexer:    leXer,
		parser:   parser,
		resolver: resolver,
	}
}

func (c Capture) Resolve(
	capturesExpr types.CaptureExpressions,
	capturesCache types.CapturedStates,
	state types.NMState) (types.CapturedStates, error) {
	if len(capturesExpr) == 0 || len(state) == 0 && len(capturesCache) == 0 {
		return nil, nil
	}

	capturesState := filterCacheBasedOnExprCaptures(capturesCache, capturesExpr)
	capturesExpr = filterOutExprBasedOnCachedCaptures(capturesExpr, capturesCache)

	astPool := types.CaptureASTPool{}
	for capID, capExpr := range capturesExpr {
		tokens, err := c.lexer.Lex(capExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
		}

		astRoot, err := c.parser.Parse(capExpr, tokens)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
		}

		astPool[capID] = astRoot
	}

	resolvedCapturedStates, err := c.resolver.Resolve(capturesExpr, astPool, state, capturesState)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve capture expression, err: %v", err)
	}

	return resolvedCapturedStates, nil
}

func filterOutExprBasedOnCachedCaptures(capturesExpr types.CaptureExpressions,
	capturesCache types.CapturedStates) types.CaptureExpressions {
	for capID := range capturesCache {
		delete(capturesExpr, capID)
	}
	return capturesExpr
}

func filterCacheBasedOnExprCaptures(capsState types.CapturedStates,
	capsExpr types.CaptureExpressions) types.CapturedStates {
	caps := types.CapturedStates{}

	for capID := range capsExpr {
		if capState, ok := capsState[capID]; ok {
			caps[capID] = capState
		}
	}
	return caps
}
