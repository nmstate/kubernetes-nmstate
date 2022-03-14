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
	"fmt"
	"strconv"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/ast"
	"github.com/nmstate/nmpolicy/nmpolicy/internal/expression"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer"
)

type Parser struct{}

type parser struct {
	expression      string
	tokens          []lexer.Token
	currentTokenIdx int
	lastNode        *ast.Node
	pipedInNode     *ast.Node
}

func New() Parser {
	return Parser{}
}

func newParser(expr string, tokens []lexer.Token) *parser {
	return &parser{expression: expr, tokens: tokens}
}

func (Parser) Parse(expr string, tokens []lexer.Token) (ast.Node, error) {
	return newParser(expr, tokens).Parse()
}

func (p *parser) Parse() (ast.Node, error) {
	node, err := p.parse()
	if err != nil {
		return ast.Node{}, expression.WrapError(err, p.expression, p.currentToken().Position)
	}
	return node, nil
}

func (p *parser) parse() (ast.Node, error) {
	for {
		if p.currentToken() == nil {
			return ast.Node{}, nil
		} else if p.currentToken().Type == lexer.EOF {
			break
		} else if p.currentToken().Type == lexer.STRING {
			if err := p.parseString(); err != nil {
				return ast.Node{}, err
			}
		} else if p.currentToken().Type == lexer.IDENTITY {
			if err := p.parsePath(); err != nil {
				return ast.Node{}, err
			}
		} else if p.currentToken().Type == lexer.EQFILTER {
			if err := p.parseEqFilter(); err != nil {
				return ast.Node{}, err
			}
		} else if p.currentToken().Type == lexer.REPLACE {
			if err := p.parseReplace(); err != nil {
				return ast.Node{}, err
			}
		} else if p.currentToken().Type == lexer.PIPE {
			if err := p.parsePipe(); err != nil {
				return ast.Node{}, err
			}
		} else {
			return ast.Node{}, invalidExpressionError(fmt.Sprintf("unexpected token `%+v`", p.currentToken().Literal))
		}
		p.nextToken()
	}
	if p.pipedInNode != nil {
		return ast.Node{}, invalidPipeError("missing pipe out expression")
	}
	return p.lastEmitedNode(), nil
}

func (p *parser) lastEmitedNode() ast.Node {
	if p.lastNode == nil {
		return ast.Node{}
	}
	return *p.lastNode
}

func (p *parser) nextToken() {
	if len(p.tokens) == 0 {
		return
	}
	if p.currentTokenIdx >= len(p.tokens)-1 {
		p.currentTokenIdx = len(p.tokens) - 1
	} else {
		p.currentTokenIdx++
	}
}

func (p *parser) prevToken() {
	if len(p.tokens) == 0 {
		return
	}
	if p.currentTokenIdx > 0 {
		p.currentTokenIdx--
	}
	if p.currentTokenIdx >= len(p.tokens)-1 {
		p.currentTokenIdx = len(p.tokens) - 1
	}
}

func (p *parser) currentToken() *lexer.Token {
	if len(p.tokens) == 0 || p.currentTokenIdx >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.currentTokenIdx]
}

func (p *parser) parseIdentity() error {
	p.lastNode = &ast.Node{
		Meta:     ast.Meta{Position: p.currentToken().Position},
		Terminal: ast.Terminal{Identity: &p.currentToken().Literal},
	}
	return nil
}

func (p *parser) parseString() error {
	p.lastNode = &ast.Node{
		Meta:     ast.Meta{Position: p.currentToken().Position},
		Terminal: ast.Terminal{Str: &p.currentToken().Literal},
	}
	return nil
}

func (p *parser) parseNumber() error {
	number, err := strconv.Atoi(p.currentToken().Literal)
	if err != nil {
		return err
	}
	p.lastNode = &ast.Node{
		Meta:     ast.Meta{Position: p.currentToken().Position},
		Terminal: ast.Terminal{Number: &number},
	}
	return nil
}

func (p *parser) parsePath() error {
	if err := p.parseIdentity(); err != nil {
		return err
	}
	operator := &ast.Node{
		Meta: ast.Meta{Position: p.currentToken().Position},
		Path: &ast.VariadicOperator{*p.lastNode},
	}
	for {
		p.nextToken()
		if p.currentToken().Type == lexer.DOT {
			p.nextToken()
			if p.currentToken().Type == lexer.IDENTITY {
				if err := p.parseIdentity(); err != nil {
					return err
				}
			} else if p.currentToken().Type == lexer.NUMBER {
				if err := p.parseNumber(); err != nil {
					return wrapWithInvalidPathError(err)
				}
			} else {
				return invalidPathError("missing identity or number after dot")
			}
			path := append(*operator.Path, *p.lastNode)
			operator.Path = &path
		} else if p.currentToken().Type != lexer.EOF && !p.currentToken().Type.IsOperator() {
			return invalidPathError("missing dot")
		} else {
			// Token has not being consumed let's go back.
			p.prevToken()
			break
		}
	}
	p.lastNode = operator
	return nil
}

func (p *parser) parseEqFilter() error {
	operator := &ast.Node{
		Meta:     ast.Meta{Position: p.currentToken().Position},
		EqFilter: &ast.TernaryOperator{},
	}
	if err := p.fillInTernaryOperator(operator.EqFilter); err != nil {
		return wrapWithInvalidEqualityFilterError(err)
	}
	p.lastNode = operator
	return nil
}

func (p *parser) parseReplace() error {
	operator := &ast.Node{
		Meta:    ast.Meta{Position: p.currentToken().Position},
		Replace: &ast.TernaryOperator{},
	}
	if err := p.fillInTernaryOperator(operator.Replace); err != nil {
		return wrapWithInvalidReplaceError(err)
	}
	p.lastNode = operator
	return nil
}

func (p *parser) fillInTernaryOperator(operator *ast.TernaryOperator) error {
	if p.lastNode == nil {
		return fmt.Errorf("missing left hand argument")
	}
	if p.lastNode.Path == nil {
		return fmt.Errorf("left hand argument is not a path")
	}

	p.fillInPipedInOrCurrentState(&operator[0])

	operator[1] = *p.lastNode

	p.nextToken()
	if p.currentToken().Type == lexer.STRING {
		if err := p.parseString(); err != nil {
			return err
		}
		operator[2] = *p.lastNode
	} else if p.currentToken().Type == lexer.IDENTITY {
		err := p.parsePath()
		if err != nil {
			return err
		}
		operator[2] = *p.lastNode
	} else if p.currentToken().Type == lexer.EOF {
		return fmt.Errorf("missing right hand argument")
	} else {
		return fmt.Errorf("right hand argument is not a string or identity")
	}
	return nil
}

func (p *parser) fillInPipedInOrCurrentState(node *ast.Node) {
	if p.pipedInNode != nil {
		*node = *p.pipedInNode
		p.pipedInNode = nil
	} else {
		node.Terminal = ast.CurrentStateIdentity()
	}
}

func (p *parser) parsePipe() error {
	if p.lastNode == nil {
		return invalidPipeError("missing pipe in expression")
	}
	if p.lastNode.Path == nil {
		return invalidPipeError("only paths can be piped in")
	}
	p.pipedInNode = p.lastNode
	return nil
}
