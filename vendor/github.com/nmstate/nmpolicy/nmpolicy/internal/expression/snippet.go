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
	"strings"
)

// Snippet returns a string containing src and a pointer at pos.
// Example of str "123456" and pos "4":
//
// | 123456
// | ...^
func snippet(expression string, pos int) string {
	if expression == "" {
		return ""
	}

	if pos >= len(expression) {
		pos = len(expression) - 1
	}

	marker := strings.Builder{}
	for i := 0; i < pos; i++ {
		marker.WriteString(".")
	}
	marker.WriteString("^")
	return fmt.Sprintf("| %s\n| %s", expression, marker.String())
}
