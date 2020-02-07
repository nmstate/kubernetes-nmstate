package runner

import (
	"strings"

	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
)

func nmstatePods() ([]string, error) {
	output, err := cmd.Kubectl("get", "pod", "-n", framework.Global.Namespace, "--no-headers=true", "-o", "custom-columns=:metadata.name")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	names := strings.Split(strings.TrimSpace(output), "\n")
	return names, err
}

func RunAtPods(arguments ...string) {
	nmstatePods, err := nmstatePods()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	for _, nmstatePod := range nmstatePods {
		exec := []string{"exec", "-n", framework.Global.Namespace, nmstatePod, "--"}
		execArguments := append(exec, arguments...)
		_, err := cmd.Kubectl(execArguments...)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}
}
