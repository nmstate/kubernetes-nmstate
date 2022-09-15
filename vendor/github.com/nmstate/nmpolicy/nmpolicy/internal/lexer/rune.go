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

import (
	"strings"
	"unicode"

	"github.com/nmstate/nmpolicy/nmpolicy/internal/lexer/scanner"
)

func (l *lexer) isDigit() bool {
	return unicode.IsDigit(l.scn.Rune())
}

func (l *lexer) isSpace() bool {
	return unicode.IsSpace(l.scn.Rune())
}

func (l *lexer) isEOF() bool {
	return l.scn.Rune() == scanner.EOF
}

func (l *lexer) isString() bool {
	return strings.ContainsRune(`"'`, l.scn.Rune())
}

func (l *lexer) isLetter() bool {
	return unicode.IsLetter(l.scn.Rune())
}

func (l *lexer) isDot() bool {
	return l.scn.Rune() == '.'
}

func (l *lexer) isEqual() bool {
	return l.scn.Rune() == '='
}

func (l *lexer) isColon() bool {
	return l.scn.Rune() == ':'
}

func (l *lexer) isPlus() bool {
	return l.scn.Rune() == '+'
}

func (l *lexer) isPipe() bool {
	return l.scn.Rune() == '|'
}

func (l *lexer) isDelimiter() bool {
	return l.isEOF() || l.isSpace() || l.isDot() || l.isEqual() || l.isColon() || l.isPlus() || l.isPipe()
}
