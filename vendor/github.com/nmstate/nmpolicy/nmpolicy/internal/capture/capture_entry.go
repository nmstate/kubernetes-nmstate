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

	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/parser"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/resolver"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/types"
)

type CaptureEntry struct {
	capturedStates types.CapturedStates
	lexer          Lexer
	parser         Parser
	resolver       Resolver
}

func NewCaptureEntryWithLexerParserResolver(capturedStates types.CapturedStates,
	l Lexer, p Parser, r Resolver) (CaptureEntry, error) {
	return CaptureEntry{
		capturedStates: capturedStates,
		lexer:          l,
		parser:         p,
		resolver:       r,
	}, nil
}

func NewCaptureEntry(capturedStates types.CapturedStates) (CaptureEntry, error) {
	return NewCaptureEntryWithLexerParserResolver(capturedStates, lexer.New(), parser.New(), resolver.New())
}

func (c CaptureEntry) ResolveCaptureEntryPath(
	captureEntryPathExpression string) (interface{}, error) {
	captureEntryPathTokens, err := c.lexer.Lex(captureEntryPathExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve capture entry path expression: %v", err)
	}

	captureEntryPathAST, err := c.parser.Parse(captureEntryPathTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve capture entry path expression: %v", err)
	}

	resolvedCaptureEntryPath, err := c.resolver.ResolveCaptureEntryPath(captureEntryPathAST, c.capturedStates)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve capture entry path expression: %v", err)
	}

	return resolvedCaptureEntryPath, nil
}
