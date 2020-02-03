package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
)

func run(command string, quiet bool, arguments ...string) (string, error) {
	cmd := exec.Command(command, arguments...)
	if !quiet {
		GinkgoWriter.Write([]byte(command + " " + strings.Join(arguments, " ") + "\n"))
	}
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if !quiet {
		GinkgoWriter.Write([]byte(fmt.Sprintf("stdout: %.500s...\n, stderr %s\n", stdout.String(), stderr.String())))
	}
	return stdout.String(), err
}

func runAtNodeWithExtras(node string, quiet bool, command ...string) (string, error) {
	ssh_command := []string{node, "--"}
	ssh_command = append(ssh_command, command...)
	output, err := run("./kubevirtci/cluster-up/ssh.sh", quiet, ssh_command...)
	// Remove first two lines from output, ssh.sh add garbage there
	outputLines := strings.Split(output, "\n")
	output = strings.Join(outputLines[2:], "\n")
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
