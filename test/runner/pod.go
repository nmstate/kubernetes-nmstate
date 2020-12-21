package runner

import (
	"strings"

	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

func nmstateHandlerPods() ([]string, error) {
	output, err := cmd.Kubectl("get", "pod", "-n", testenv.OperatorNamespace, "--no-headers=true", "-o", "custom-columns=:metadata.name", "-l", "component=kubernetes-nmstate-handler")
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	names := strings.Split(strings.TrimSpace(output), "\n")
	return names, err
}

func runAtPod(pod string, arguments ...string) string {
	exec := []string{"exec", "-n", testenv.OperatorNamespace, pod, "--"}
	execArguments := append(exec, arguments...)
	output, err := cmd.Kubectl(execArguments...)
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	return output
}

func runAtPods(pods []string, arguments ...string) {
	for _, pod := range pods {
		runAtPod(pod, arguments...)
	}
}

func RunAtFirstHandlerPod(arguments ...string) string {
	handlerPods, err := nmstateHandlerPods()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, handlerPods).ToNot(BeEmpty())
	return runAtPod(handlerPods[0], arguments...)
}

func RunAtHandlerPods(arguments ...string) {
	handlerPods, err := nmstateHandlerPods()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	runAtPods(handlerPods, arguments...)
}
