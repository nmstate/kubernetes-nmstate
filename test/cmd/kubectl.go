package cmd

import (
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

func Kubectl(arguments ...string) (string, error) {
	kubectl := environment.GetVarWithDefault("KUBECTL", "./cluster/kubectl.sh")
	return Run(kubectl, false, arguments...)
}
