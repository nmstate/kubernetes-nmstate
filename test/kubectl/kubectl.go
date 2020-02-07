package kubectl

import (
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
)

func Kubectl(arguments ...string) (string, error) {
	return cmd.Run("./kubevirtci/cluster-up/kubectl.sh", false, arguments...)
}
