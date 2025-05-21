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
	"context"
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

const nmstateCommand = "nmstatectl"

func nmstatectlWithInput(ctx context.Context, arguments []string, input string) (string, error) {
	cmd := exec.CommandContext(ctx, nmstateCommand, arguments...)
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("failed to create pipe for writing into %s: %v", nmstateCommand, err)
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
		return "", fmt.Errorf(
			"failed to execute %s %s: '%v' '%s' '%s'",
			nmstateCommand,
			strings.Join(arguments, " "),
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	return stdout.String(), nil
}

func nmstatectl(ctx context.Context, arguments []string) (string, error) {
	return nmstatectlWithInput(ctx, arguments, "")
}

func Show() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return nmstatectl(ctx, []string{"show"})
}

func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	var setDoneCh = make(chan struct{})
	defer close(setDoneCh)

	setOutput, err := nmstatectlWithInput(context.TODO(),
		[]string{"apply", "-v", "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds()))},
		string(desiredState.Raw),
	)
	return setOutput, err
}

func Commit() (string, error) {
	return nmstatectl(context.TODO(), []string{"commit"})
}

func Rollback() error {
	_, err := nmstatectl(context.TODO(), []string{"rollback"})
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

func (s *Stats) Subtract(statsToSubstract *Stats) Stats {
	// Clone the features
	result := Stats{Features: map[string]bool{}}
	for k, v := range s.Features {
		result.Features[k] = v
	}

	// Subtract the selected ones
	for f := range statsToSubstract.Features {
		delete(result.Features, f)
	}
	return result
}

func Statistic(desiredState nmstate.State) (*Stats, error) {
	statsOutput, err := nmstatectlWithInput(context.TODO(),
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
	out, err := nmstatectl(context.TODO(), args)
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
