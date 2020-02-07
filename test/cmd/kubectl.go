package cmd

func Kubectl(arguments ...string) (string, error) {
	return Run("./kubevirtci/cluster-up/kubectl.sh", false, arguments...)
}
