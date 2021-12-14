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

package scanner

import (
	"io"
)

const (
	EOF rune = -1
)

type Scanner struct {
	reader      io.RuneScanner
	currentRune rune
	prevRune    rune
	pos         int
}

func New(reader io.RuneScanner) *Scanner {
	return &Scanner{
		reader: reader,
	}
}

func (s *Scanner) Next() error {
	rn, err := s.next()
	if err != nil {
		return err
	}
	s.prevRune = s.currentRune
	s.currentRune = rn
	if rn != EOF {
		s.pos++
	}
	return nil
}

func (s *Scanner) Rune() rune {
	return s.currentRune
}

func (s *Scanner) Prev() error {
	if err := s.reader.UnreadRune(); err != nil {
		return err
	}
	s.currentRune = s.prevRune
	s.pos--
	return nil
}

func (s *Scanner) Position() int {
	return s.pos - 1
}

func (s *Scanner) next() (rune, error) {
	rn, _, err := s.reader.ReadRune()
	if err != nil {
		if err == io.EOF {
			return EOF, nil
		}
		return EOF, err
	}
	return rn, nil
}
