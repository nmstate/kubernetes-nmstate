package cmd

func Kubectl(arguments ...string) (string, error) {
	return Run("./cluster/kubectl.sh", false, arguments...)
}
