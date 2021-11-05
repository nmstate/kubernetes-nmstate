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

package parser

import (
	"reflect"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
)

type Parser struct{}

func New() Parser {
	return Parser{}
}

func (p Parser) Parse(tokens []lexer.Token) (ast.Node, error) {
	tokenRoutes := lexer.Token{Position: 0, Type: lexer.IDENTITY, Literal: "routes"}
	tokenRunning := lexer.Token{Position: 7, Type: lexer.IDENTITY, Literal: "running"}
	tokenDestination := lexer.Token{Position: 15, Type: lexer.IDENTITY, Literal: "destination"}
	tokenDefaultGw := lexer.Token{Position: 28, Type: lexer.STRING, Literal: "0.0.0.0/0"}
	tokenEqFilter := lexer.Token{Position: 26, Type: lexer.EQFILTER, Literal: "=="}

	if reflect.DeepEqual(tokens, []lexer.Token{
		tokenRoutes,
		{Position: 6, Type: lexer.DOT, Literal: "."},
		tokenRunning,
		{Position: 14, Type: lexer.DOT, Literal: "."},
		tokenDestination,
		tokenEqFilter,
		tokenDefaultGw,
		{Position: 38, Type: lexer.EOF, Literal: ""},
	}) {
		return ast.Node{
			Meta: ast.Meta{Position: tokenEqFilter.Position},
			EqFilter: &ast.TernaryOperator{
				ast.Node{
					Meta:     ast.Meta{Position: 0},
					Terminal: ast.CurrentStateIdentity()},
				ast.Node{
					Meta: ast.Meta{Position: 0},
					Path: &ast.VariadicOperator{
						ast.Node{
							Meta:     ast.Meta{Position: tokenRoutes.Position},
							Terminal: ast.Terminal{Identity: &tokenRoutes.Literal},
						},
						ast.Node{
							Meta:     ast.Meta{Position: tokenRunning.Position},
							Terminal: ast.Terminal{Identity: &tokenRunning.Literal},
						},
						ast.Node{
							Meta:     ast.Meta{Position: tokenDestination.Position},
							Terminal: ast.Terminal{Identity: &tokenDestination.Literal},
						},
					},
				},
				ast.Node{
					Meta:     ast.Meta{Position: tokenDefaultGw.Position},
					Terminal: ast.Terminal{String: &tokenDefaultGw.Literal},
				},
			},
		}, nil
	}
	return ast.Node{}, nil
}
