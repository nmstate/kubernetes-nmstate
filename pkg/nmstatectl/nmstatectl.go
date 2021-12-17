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
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

const nmstateCommand = "nmstatectl"

func nmstatectlWithInput(arguments []string, input string) (string, error) {
	cmd := exec.Command(nmstateCommand, arguments...)
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
		return "", fmt.Errorf("failed to execute %s %s: '%v' '%s' '%s'", nmstateCommand, strings.Join(arguments, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func nmstatectl(arguments []string) (string, error) {
	return nmstatectlWithInput(arguments, "")
}

func Show() (string, error) {
	return nmstatectl([]string{"show"})
}

func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	var setDoneCh = make(chan struct{})
	go setUnavailableUp(setDoneCh)
	defer close(setDoneCh)

	setOutput, err := nmstatectlWithInput([]string{"set", "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds()))}, string(desiredState.Raw))
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
