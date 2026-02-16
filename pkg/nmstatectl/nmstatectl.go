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
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var debugMode bool

const nmstateCommand = "nmstatectl"

// execCommand is a variable that can be overridden in tests to capture command execution
var execCommand = exec.Command

func SetDebugMode(debug bool) {
	debugMode = debug
}

func nmstatectlWithInputAndOutputs(arguments []string, input string, stdout, stderr io.Writer) error {
	cmd := execCommand(nmstateCommand, arguments...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create pipe for writing into %s: %v", nmstateCommand, err)
		}
		go func() {
			defer stdin.Close()
			_, err = io.WriteString(stdin, input)
			if err != nil {
				fmt.Printf("failed to write input into stdin: %v\n", err)
			}
		}()
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"failed to execute %s %s: %w",
			nmstateCommand,
			strings.Join(arguments, " "),
			err,
		)
	}
	return nil
}

func nmstatectlWithInput(arguments []string, input string) (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := nmstatectlWithInputAndOutputs(arguments, input, stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("%s, %s: %w", stdout.String(), stderr.String(), err)
	}
	return stdout.String(), nil
}

func nmstatectl(arguments []string) (string, error) {
	return nmstatectlWithInput(arguments, "")
}

func ShowWithArgumentsAndOutputs(arguments []string, stdout, stderr io.Writer) error {
	return nmstatectlWithInputAndOutputs(append([]string{"show"}, arguments...), "", stdout, stderr)
}

func Show() (string, error) {
	return nmstatectl([]string{"show"})
}

func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	var setDoneCh = make(chan struct{})
	defer close(setDoneCh)

	args := []string{"apply"}
	if debugMode {
		args = append(args, "-vv")
	}
	args = append(args, "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds())))

	setOutput, err := nmstatectlWithInput(args, string(desiredState.Raw))
	return setOutput, err
}

func Commit() (string, error) {
	return nmstatectl([]string{"commit"})
}

func Rollback() error {
	_, err := nmstatectl([]string{"rollback"})
	if err != nil {
		return errors.Wrapf(err, "failed calling nmstatectl rollback")
	}
	return nil
}

type Stats struct {
	Features map[string]bool
}

func NewStats(features []string) *Stats {
	stats := Stats{
		Features: map[string]bool{},
	}
	for _, f := range features {
		stats.Features[f] = true
	}
	return &stats
}

func Statistic(desiredState nmstate.State) (*Stats, error) {
	statsOutput, err := nmstatectlWithInput(
		[]string{"st", "-"},
		string(desiredState.Raw),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed calling nmstatectl statistics")
	}
	stats := struct {
		Features []string `json:"features"`
	}{}
	err = yaml.Unmarshal([]byte(statsOutput), &stats)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling nmstatectl statistics")
	}
	return NewStats(stats.Features), nil
}

func Policy(policy, currentState, capturedState []byte) (desiredState, generatedCapturedState []byte, err error) {
	policyFile, err := generateFileWithContent("policy", policy)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		os.Remove(policyFile)
	}()

	args := []string{"policy"}
	if len(currentState) > 0 {
		var currentStateFile string
		currentStateFile, err = generateFileWithContent("currentState", currentState)
		if err != nil {
			return nil, nil, err
		}
		defer func() {
			os.Remove(currentStateFile)
		}()
		args = append(args, "--current", currentStateFile)
	}

	args = append(args, "--json")

	capturedStateFile, err := generateFileWithContent("capturedState", capturedState)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		os.Remove(capturedStateFile)
	}()
	args = append(args, "--output-captured", capturedStateFile)
	if len(capturedState) > 0 {
		args = append(args, "--captured", capturedStateFile)
	}

	args = append(args, policyFile)
	out, err := nmstatectl(args)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed calling nmstatectl rollback")
	}
	capturedState, err = os.ReadFile(capturedStateFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed failed reading captured state")
	}
	return []byte(out), capturedState, nil
}

func generateFileWithContent(name string, content []byte) (string, error) {
	file, err := os.CreateTemp("/tmp", name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}
