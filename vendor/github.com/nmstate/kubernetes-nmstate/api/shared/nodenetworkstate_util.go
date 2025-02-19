/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

// custom marshaling/unmarshaling that will allow to populate nmstate state
// as play yaml without the need to generate a golang struct following [1]
//
// The yaml parser we use here first convert yaml to json and then manage
// the data with the standard golang json package.
//
// [1] https://github.com/nmstate/nmstate/blob/base/libnmstate/schemas/operational-state.yaml

import (
	yaml "sigs.k8s.io/yaml"
)

// We are using behind the scenes the golang encode/json so we need to return
// json here for golang to work well, the upper yaml parser will convert it
// to yaml making nmstate yaml transparent to kubernetes-nmstate
func (t State) MarshalJSON() (output []byte, err error) {
	return yaml.YAMLToJSON([]byte(t.Raw))
}

// Bypass State parsing and directly store it as yaml string to later on
// pass it to namestatectl using it as transparet data at kubernetes-nmstate
func (t *State) UnmarshalJSON(b []byte) error {
	output, err := yaml.JSONToYAML(b)
	if err != nil {
		return err
	}
	*t = State{Raw: output}
	return nil
}

// Simple stringer for State
func (t State) String() string {
	return string(t.Raw)
}
