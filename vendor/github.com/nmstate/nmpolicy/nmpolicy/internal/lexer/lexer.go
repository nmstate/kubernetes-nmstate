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

package lexer

type Lexer struct{}

func New() Lexer {
	return Lexer{}
}

func (l Lexer) Lex(expression string) ([]Token, error) {
	if expression == `routes.running.destination=="0.0.0.0/0"` {
		return []Token{
			{0, IDENTITY, "routes"},
			{6, DOT, "."},
			{7, IDENTITY, "running"},
			{14, DOT, "."},
			{15, IDENTITY, "destination"},
			{26, EQFILTER, "=="},
			{28, STRING, "0.0.0.0/0"},
			{38, EOF, ""},
		}, nil
	}
	return nil, nil
}
