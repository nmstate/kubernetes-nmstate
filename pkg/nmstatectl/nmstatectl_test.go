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

package nmstatectl

import (
	"os/exec"
	"reflect"
	"testing"
	"time"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func TestSetCommandAndDebugMode(t *testing.T) {
	tests := []struct {
		name         string
		debugMode    bool
		expectedCmd  string
		expectedArgs []string
	}{
		{
			name:         "with debug mode",
			debugMode:    true,
			expectedCmd:  "nmstatectl",
			expectedArgs: []string{"apply", "-vv", "--no-commit", "--timeout", "120"},
		},
		{
			name:         "without debug mode",
			debugMode:    false,
			expectedCmd:  "nmstatectl",
			expectedArgs: []string{"apply", "--no-commit", "--timeout", "120"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDebugMode := debugMode
			originalExecCommand := execCommand
			defer func() {
				debugMode = originalDebugMode
				execCommand = originalExecCommand
			}()

			var capturedCmd string
			var capturedArgs []string

			execCommand = func(name string, args ...string) *exec.Cmd {
				capturedCmd = name
				capturedArgs = args
				cmd := exec.Command("echo", "mocked output")
				return cmd
			}

			timeout := 120 * time.Second
			desiredState := nmstate.State{Raw: []byte(`{"interfaces": []}`)}

			SetDebugMode(tt.debugMode)
			_, err := Set(desiredState, timeout)
			if err != nil {
				t.Errorf("Set() failed: %v", err)
			}

			if capturedCmd != tt.expectedCmd {
				t.Errorf("Expected command '%s', got '%s'", tt.expectedCmd, capturedCmd)
			}

			if !reflect.DeepEqual(capturedArgs, tt.expectedArgs) {
				t.Errorf("Arguments = %v, want %v", capturedArgs, tt.expectedArgs)
			}
		})
	}
}
