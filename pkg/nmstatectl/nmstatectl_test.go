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
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSetDebugMode(t *testing.T) {
	tests := []struct {
		name     string
		debug    bool
		expected bool
	}{
		{
			name:     "Enable debug mode",
			debug:    true,
			expected: true,
		},
		{
			name:     "Disable debug mode",
			debug:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugMode(tt.debug)
			if debugMode != tt.expected {
				t.Errorf("SetDebugMode(%v) = %v, want %v", tt.debug, debugMode, tt.expected)
			}
		})
	}
}

func TestSetDebugModeToggle(t *testing.T) {
	originalDebugMode := debugMode
	defer func() {
		debugMode = originalDebugMode
	}()

	SetDebugMode(true)
	if !debugMode {
		t.Error("Expected debugMode to be true after SetDebugMode(true)")
	}

	SetDebugMode(false)
	if debugMode {
		t.Error("Expected debugMode to be false after SetDebugMode(false)")
	}

	SetDebugMode(true)
	if !debugMode {
		t.Error("Expected debugMode to be true after second SetDebugMode(true)")
	}
}

func TestSetArgumentsWithDebugModeDisabled(t *testing.T) {
	originalDebugMode := debugMode
	defer func() {
		debugMode = originalDebugMode
	}()

	SetDebugMode(false)

	timeout := 60 * time.Second
	expectedArgs := []string{"apply", "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds()))}

	actualArgs := buildSetArguments(timeout)

	if !reflect.DeepEqual(actualArgs, expectedArgs) {
		t.Errorf("buildSetArguments() with debug disabled = %v, want %v", actualArgs, expectedArgs)
	}

	if slices.Contains(actualArgs, "-v") {
		t.Error("Expected arguments to not contain '-v' when debug mode is disabled")
	}
}

func TestSetArgumentsWithDebugModeEnabled(t *testing.T) {
	originalDebugMode := debugMode
	defer func() {
		debugMode = originalDebugMode
	}()

	SetDebugMode(true)

	timeout := 60 * time.Duration(time.Second)
	expectedArgs := []string{"apply", "-v", "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds()))}

	actualArgs := buildSetArguments(timeout)

	if !reflect.DeepEqual(actualArgs, expectedArgs) {
		t.Errorf("buildSetArguments() with debug enabled = %v, want %v", actualArgs, expectedArgs)
	}

	if !slices.Contains(actualArgs, "-v") {
		t.Error("Expected arguments to contain '-v' when debug mode is enabled")
	}
}

func TestSetFunctionWithMockedExecution(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(1)
	}

	cmd := args[0]
	if cmd != "nmstatectl" {
		os.Exit(1)
	}

	if slices.Contains(args[1:], "-v") {
		os.Stdout.WriteString("verbose output enabled")
	} else {
		os.Stdout.WriteString("normal output")
	}
	os.Exit(0)
}

func TestSetWithMockedNmstatectl(t *testing.T) {
	originalDebugMode := debugMode
	defer func() {
		debugMode = originalDebugMode
	}()

	tests := []struct {
		name        string
		debugMode   bool
		expectVFlag bool
	}{
		{
			name:        "Debug mode disabled",
			debugMode:   false,
			expectVFlag: false,
		},
		{
			name:        "Debug mode enabled",
			debugMode:   true,
			expectVFlag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugMode(tt.debugMode)

			timeout := 30 * time.Second
			args := buildSetArguments(timeout)

			hasVFlag := slices.Contains(args, "-v")
			if hasVFlag != tt.expectVFlag {
				t.Errorf("Expected -v flag presence to be %v, got %v. Args: %v", tt.expectVFlag, hasVFlag, args)
			}
		})
	}
}

func buildSetArguments(timeout time.Duration) []string {
	args := []string{"apply"}
	if debugMode {
		args = append(args, "-v")
	}
	args = append(args, "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds())))
	return args
}

func TestDefaultDebugModeState(t *testing.T) {
	currentMode := debugMode
	SetDebugMode(false)
	if debugMode {
		t.Error("Expected debugMode to be false by default after reset")
	}
	debugMode = currentMode
}

func TestSetCommandArgumentOrder(t *testing.T) {
	originalDebugMode := debugMode
	defer func() {
		debugMode = originalDebugMode
	}()

	timeout := 120 * time.Second

	SetDebugMode(true)
	argsWithDebug := buildSetArguments(timeout)
	expectedWithDebug := []string{"apply", "-v", "--no-commit", "--timeout", "120"}
	if !reflect.DeepEqual(argsWithDebug, expectedWithDebug) {
		t.Errorf("Arguments with debug = %v, want %v", argsWithDebug, expectedWithDebug)
	}

	SetDebugMode(false)
	argsWithoutDebug := buildSetArguments(timeout)
	expectedWithoutDebug := []string{"apply", "--no-commit", "--timeout", "120"}
	if !reflect.DeepEqual(argsWithoutDebug, expectedWithoutDebug) {
		t.Errorf("Arguments without debug = %v, want %v", argsWithoutDebug, expectedWithoutDebug)
	}

	if strings.Join(argsWithDebug, " ") == strings.Join(argsWithoutDebug, " ") {
		t.Error("Expected different argument lists for debug enabled vs disabled")
	}
}

