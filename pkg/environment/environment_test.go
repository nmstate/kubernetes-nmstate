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

package environment

import (
	"testing"
	"time"
)

const testEnvVar = "KUBERNETES_NMSTATE_TEST_ENV_VAR"

func TestGetEnvVarAsPositiveInt(t *testing.T) {
	const defaultVal = 5
	tests := []struct {
		name     string
		value    *string
		expected int
	}{
		{name: "unset", value: nil, expected: defaultVal},
		{name: "positive", value: new("3"), expected: 3},
		{name: "zero", value: new("0"), expected: defaultVal},
		{name: "negative", value: new("-1"), expected: defaultVal},
		{name: "not an integer", value: new("foo"), expected: defaultVal},
		{name: "empty", value: new(""), expected: defaultVal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != nil {
				t.Setenv(testEnvVar, *tt.value)
			}
			if got := GetEnvVarAsPositiveInt(testEnvVar, defaultVal); got != tt.expected {
				t.Errorf("GetEnvVarAsPositiveInt() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestGetEnvVarAsPositiveDuration(t *testing.T) {
	const defaultVal = 30 * time.Second
	tests := []struct {
		name     string
		value    *string
		expected time.Duration
	}{
		{name: "unset", value: nil, expected: defaultVal},
		{name: "positive", value: new("5"), expected: 5 * time.Second},
		{name: "zero", value: new("0"), expected: defaultVal},
		{name: "negative", value: new("-1"), expected: defaultVal},
		{name: "not an integer", value: new("1s"), expected: defaultVal},
		{name: "empty", value: new(""), expected: defaultVal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != nil {
				t.Setenv(testEnvVar, *tt.value)
			}
			if got := GetEnvVarAsPositiveDuration(testEnvVar, defaultVal); got != tt.expected {
				t.Errorf("GetEnvVarAsPositiveDuration() = %s, expected %s", got, tt.expected)
			}
		})
	}
}
