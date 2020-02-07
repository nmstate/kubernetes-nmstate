package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
)

func Run(command string, quiet bool, arguments ...string) (string, error) {
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
