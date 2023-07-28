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

package ast

import "fmt"

type Meta struct {
	Position int `json:"pos"`
}

type TernaryOperator [3]Node
type VariadicOperator []Node
type Terminal struct {
	Str      *string `json:"string,omitempty"`
	Identity *string `json:"identity,omitempty"`
	Number   *int    `json:"number,omitempty"`
	Boolean  *bool   `json:"boolean,omitempty"`
}

type Node struct {
	Meta
	EqFilter *TernaryOperator  `json:"eqfilter,omitempty"`
	NeFilter *TernaryOperator  `json:"nefilter,omitempty"`
	Replace  *TernaryOperator  `json:"replace,omitempty"`
	Path     *VariadicOperator `json:"path,omitempty"`
	Terminal
}

func (n Node) String() string {
	if n.EqFilter != nil {
		return fmt.Sprintf("EqFilter(%s)", *n.EqFilter)
	}
	if n.NeFilter != nil {
		return fmt.Sprintf("NeFilter(%s)", *n.NeFilter)
	}
	if n.Replace != nil {
		return fmt.Sprintf("Replace(%s)", *n.Replace)
	}
	if n.Path != nil {
		return fmt.Sprintf("Path=%s", *n.Path)
	}
	return n.Terminal.String()
}

func (t Terminal) String() string {
	if t.Str != nil {
		return fmt.Sprintf("String=%s", *t.Str)
	}
	if t.Identity != nil {
		return fmt.Sprintf("Identity=%s", *t.Identity)
	}
	if t.Number != nil {
		return fmt.Sprintf("Number=%d", *t.Number)
	}
	if t.Boolean != nil {
		return fmt.Sprintf("Boolean=%t", *t.Boolean)
	}
	return ""
}
