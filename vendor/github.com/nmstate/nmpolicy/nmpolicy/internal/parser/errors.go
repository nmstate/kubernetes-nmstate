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
	"strings"
)

type parserError struct {
	prefix string
	inner  error
	msg    string
}

func (p parserError) Unwrap() error {
	return p.inner
}

func (p parserError) Error() string {
	errorMsg := strings.Builder{}
	errorMsg.WriteString(p.prefix)
	errorMsg.WriteString(": ")
	if p.inner != nil {
		errorMsg.WriteString(p.inner.Error())
	} else {
		errorMsg.WriteString(p.msg)
	}
	return errorMsg.String()
}

const invalidPathErrorPrefix = "invalid path"

func wrapWithInvalidPathError(err error) *parserError {
	return &parserError{
		prefix: invalidPathErrorPrefix,
		inner:  err,
	}
}

func invalidPathError(msg string) *parserError {
	return &parserError{
		prefix: invalidPathErrorPrefix,
		msg:    msg,
	}
}

func wrapWithInvalidEqualityFilterError(err error) *parserError {
	return &parserError{
		prefix: "invalid equality filter",
		inner:  err,
	}
}

func wrapWithInvalidInequalityFilterError(err error) *parserError {
	return &parserError{
		prefix: "invalid inequality filter",
		inner:  err,
	}
}

func wrapWithInvalidReplaceError(err error) *parserError {
	return &parserError{
		prefix: "invalid replace",
		inner:  err,
	}
}

func invalidExpressionError(msg string) *parserError {
	return &parserError{
		prefix: "invalid expression",
		msg:    msg,
	}
}

func invalidPipeError(msg string) *parserError {
	return &parserError{
		prefix: "invalid pipe",
		msg:    msg,
	}
}
