package cmd

import (
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

func Kubectl(arguments ...string) (string, error) {
	kubectl := environment.GetVarWithDefault("KUBECTL", "./kubevirtci/cluster-up/kubectl.sh")
	return Run(kubectl, false, arguments...)
}
