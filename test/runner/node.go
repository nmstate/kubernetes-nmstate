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

package runner

import (
	"strings"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

func runAtNodeWithExtras(node string, quiet bool, command ...string) (string, error) {
	ssh := environment.GetVarWithDefault("SSH", "./cluster/ssh.sh")
	sshCommand := []string{node, "--"}
	sshCommand = append(sshCommand, command...)
	output, err := cmd.Run(ssh, quiet, sshCommand...)
	// Remove first two lines from output, ssh.sh add garbage there
	outputLines := strings.Split(output, "\n")
	if len(outputLines) > 2 {
		output = strings.Join(outputLines[2:], "\n")
	}
	return output, err
}

func RunQuietAtNode(node string, command ...string) (string, error) {
	return runAtNodeWithExtras(node, true, command...)
}

func RunAtNode(node string, command ...string) (string, error) {
	return runAtNodeWithExtras(node, false, command...)
}

func RunAtNodes(nodes []string, command ...string) (outputs []string, errs []error) {
	for _, node := range nodes {
		output, err := RunAtNode(node, command...)
		outputs = append(outputs, output)
		errs = append(errs, err)
	}
	return outputs, errs
}
