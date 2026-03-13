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
	"context"
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
			originalKernelMode := kernelMode
			originalExecCommand := execCommand
			defer func() {
				debugMode = originalDebugMode
				kernelMode = originalKernelMode
				execCommand = originalExecCommand
			}()

			var capturedCmd string
			var capturedArgs []string

			execCommand = func(name string, args ...string) *exec.Cmd {
				capturedCmd = name
				capturedArgs = args
				cmd := exec.CommandContext(context.TODO(), "echo", "mocked output")
				return cmd
			}

			timeout := 120 * time.Second
			desiredState := nmstate.State{Raw: []byte(`{"interfaces": []}`)}

			SetDebugMode(tt.debugMode)
			SetKernelMode(false)
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

func TestSetKernelMode(t *testing.T) {
	tests := []struct {
		name         string
		debugMode    bool
		kernelMode   bool
		expectedArgs []string
	}{
		{
			name:         "kernel mode without debug",
			debugMode:    false,
			kernelMode:   true,
			expectedArgs: []string{"apply", "-k"},
		},
		{
			name:         "kernel mode with debug",
			debugMode:    true,
			kernelMode:   true,
			expectedArgs: []string{"apply", "-vv", "-k"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDebugMode := debugMode
			originalKernelMode := kernelMode
			originalExecCommand := execCommand
			defer func() {
				debugMode = originalDebugMode
				kernelMode = originalKernelMode
				execCommand = originalExecCommand
			}()

			var capturedArgs []string
			execCommand = func(name string, args ...string) *exec.Cmd {
				capturedArgs = args
				return exec.CommandContext(context.TODO(), "echo", "mocked output")
			}

			SetDebugMode(tt.debugMode)
			SetKernelMode(tt.kernelMode)

			desiredState := nmstate.State{Raw: []byte(`{"interfaces": []}`)}
			_, err := Set(desiredState, 120*time.Second)
			if err != nil {
				t.Errorf("Set() failed: %v", err)
			}

			if !reflect.DeepEqual(capturedArgs, tt.expectedArgs) {
				t.Errorf("Arguments = %v, want %v", capturedArgs, tt.expectedArgs)
			}
		})
	}
}

func TestShowKernelMode(t *testing.T) {
	tests := []struct {
		name         string
		kernelMode   bool
		expectedArgs []string
	}{
		{
			name:         "show without kernel mode",
			kernelMode:   false,
			expectedArgs: []string{"show"},
		},
		{
			name:         "show with kernel mode",
			kernelMode:   true,
			expectedArgs: []string{"show", "-k"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalKernelMode := kernelMode
			originalExecCommand := execCommand
			defer func() {
				kernelMode = originalKernelMode
				execCommand = originalExecCommand
			}()

			var capturedArgs []string
			execCommand = func(name string, args ...string) *exec.Cmd {
				capturedArgs = args
				return exec.CommandContext(context.TODO(), "echo", "mocked output")
			}

			SetKernelMode(tt.kernelMode)
			_, err := Show()
			if err != nil {
				t.Errorf("Show() failed: %v", err)
			}

			if !reflect.DeepEqual(capturedArgs, tt.expectedArgs) {
				t.Errorf("Arguments = %v, want %v", capturedArgs, tt.expectedArgs)
			}
		})
	}
}

func TestCommitKernelMode(t *testing.T) {
	originalKernelMode := kernelMode
	originalExecCommand := execCommand
	defer func() {
		kernelMode = originalKernelMode
		execCommand = originalExecCommand
	}()

	commandExecuted := false
	execCommand = func(name string, args ...string) *exec.Cmd {
		commandExecuted = true
		return exec.CommandContext(context.TODO(), "echo", "mocked output")
	}

	SetKernelMode(true)
	output, err := Commit()
	if err != nil {
		t.Errorf("Commit() failed: %v", err)
	}
	if commandExecuted {
		t.Error("Commit() should not execute a command in kernel mode")
	}
	if output != "commit skipped (kernel mode)" {
		t.Errorf("Commit() output = %q, want %q", output, "commit skipped (kernel mode)")
	}
}

func TestRollbackKernelMode(t *testing.T) {
	originalKernelMode := kernelMode
	originalExecCommand := execCommand
	defer func() {
		kernelMode = originalKernelMode
		execCommand = originalExecCommand
	}()

	commandExecuted := false
	execCommand = func(name string, args ...string) *exec.Cmd {
		commandExecuted = true
		return exec.CommandContext(context.TODO(), "echo", "mocked output")
	}

	SetKernelMode(true)
	err := Rollback()
	if err != nil {
		t.Errorf("Rollback() failed: %v", err)
	}
	if commandExecuted {
		t.Error("Rollback() should not execute a command in kernel mode")
	}
}
