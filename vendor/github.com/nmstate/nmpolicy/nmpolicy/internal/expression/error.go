/*
 * Copyright 2001 NMPolicy Authors.
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

package expression

import (
	"fmt"
)

// WrapError construct a new error wrapping the error and decorating it
// with the expression and a pointer at the position specified.
func WrapError(err error, expression string, pos int) error {
	return fmt.Errorf("%w\n%s", err, snippet(expression, pos))
}

// DecorateError construct a new error including the error and decorating it
// with the expression and a pointer at the position specified.
func DecorateError(err error, expression string, pos int) error {
	return fmt.Errorf("%s\n%s", err, snippet(expression, pos))
}
