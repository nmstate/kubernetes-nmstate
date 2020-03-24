package runner

import (
	"strings"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

func runAtNodeWithExtras(node string, quiet bool, command ...string) (string, error) {
	ssh_command := []string{node, "--"}
	ssh_command = append(ssh_command, command...)
	ssh = environment.GetVarWithDefault("SSH", "./kubevirtci/cluster-up/ssh.sh")
	output, err := cmd.Run(ssh, quiet, ssh_command...)
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
