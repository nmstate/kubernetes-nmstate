package runner

import (
	"strings"

	. "github.com/onsi/gomega"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
)

func nmstateHandlerPods() ([]string, error) {
	output, err := cmd.Kubectl("get", "pod", "-n", framework.Global.Namespace, "--no-headers=true", "-o", "custom-columns=:metadata.name", "-l", "component=kubernetes-nmstate-handler")
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	names := strings.Split(strings.TrimSpace(output), "\n")
	return names, err
}

func runAtPods(pods []string, arguments ...string) {
	for _, pod := range pods {
		exec := []string{"exec", "-n", framework.Global.Namespace, pod, "--"}
		execArguments := append(exec, arguments...)
		_, err := cmd.Kubectl(execArguments...)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())
	}
}
func RunAtHandlerPods(arguments ...string) {
	handlerPods, err := nmstateHandlerPods()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	runAtPods(handlerPods, arguments...)
}
