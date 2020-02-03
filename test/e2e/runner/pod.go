package e2e

import (
	"strings"

	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

func kubectl(arguments ...string) (string, error) {
	return run("./kubevirtci/cluster-up/kubectl.sh", false, arguments...)
}

func nmstatePods() ([]string, error) {
	output, err := kubectl("get", "pod", "-n", framework.Global.Namespace, "--no-headers=true", "-o", "custom-columns=:metadata.name")
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
		_, err := kubectl(execArguments...)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}
}
