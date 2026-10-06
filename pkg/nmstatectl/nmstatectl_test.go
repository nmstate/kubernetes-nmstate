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
	"errors"
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
				cmd := exec.CommandContext(context.TODO(), "echo", "mocked output")
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

func TestNmstatectlWithTimeout(t *testing.T) {
	originalExecCommandContext := execCommandContext
	defer func() { execCommandContext = originalExecCommandContext }()

	t.Run("returns output on success", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "state")
		}
		out, err := nmstatectlWithTimeout(context.Background(), time.Second, []string{"show"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "state\n" {
			t.Errorf("unexpected output %q", out)
		}
	})

	t.Run("returns ErrTimeout and reaps the process when it hangs", func(t *testing.T) {
		var cmd *exec.Cmd
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cmd = exec.CommandContext(ctx, "sleep", "30")
			return cmd
		}
		start := time.Now()
		_, err := nmstatectlWithTimeout(context.Background(), 200*time.Millisecond, []string{"show"})
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("expected ErrTimeout, got %v", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("timeout not honored, took %s", elapsed)
		}
		// ProcessState is only set once Wait() has reaped the process, so no
		// zombie is left behind.
		if cmd.ProcessState == nil {
			t.Error("process was not reaped")
		}
	})

	t.Run("returns a non-timeout error on failure", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		}
		_, err := nmstatectlWithTimeout(context.Background(), time.Second, []string{"show"})
		if err == nil || errors.Is(err, ErrTimeout) {
			t.Fatalf("expected non-timeout error, got %v", err)
		}
	})
}

func TestShowKernelLoopbackArguments(t *testing.T) {
	originalExecCommandContext := execCommandContext
	defer func() { execCommandContext = originalExecCommandContext }()

	var capturedArgs []string
	execCommandContext = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		capturedArgs = args
		return exec.CommandContext(ctx, "true")
	}
	if err := ShowKernelLoopback(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(capturedArgs, []string{"show", "-k", "lo"}) {
		t.Errorf("unexpected arguments %v", capturedArgs)
	}
}
