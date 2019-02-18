package utils

import (
	"fmt"
	"os"

	"github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// ValidateNodeName check if the current host is a k8s node
func ValidateNodeName(cfg *restclient.Config, nodeName string) bool {
	clientset, err := k8sclient.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("Error building k8s client: %v\n", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing all nodes: %v\n", err)
	}

	for _, node := range nodes.Items {
		if nodeName == node.GetName() {
			return true
		}
	}
	return false
}

// IsStateApplicable check if the state should be applied to current node
func IsStateApplicable(cfg *restclient.Config, state *v1.NodeNetworkState, nodeName string) bool {
	if nodeName == state.Spec.NodeName {
		// node name validation is optional
		if cfg != nil && !ValidateNodeName(cfg, nodeName) {
			fmt.Printf("Warning: hostname '%s' was not found to be a valid node name\n", nodeName)
		}

		if state.Name != state.Spec.NodeName {
			fmt.Printf("Warning: resource name '%s' does not match hostname '%s'\n", state.Name, nodeName)
		}
		return true
	}

	return false
}

// GetHostName return the host name from the input
// if not set, it tries to read from an k8s based on
// env variable holding the pod's name, and if not possible either
// tries to read it from the OS
func GetHostName(hostname string, cfg *restclient.Config, ns string) string {
	if hostname != "" {
		return hostname
	}

	cause := "missing POD_NAME env variable"
	podName := os.Getenv("POD_NAME")
	if cfg != nil && podName != "" {
		clientset, err := k8sclient.NewForConfig(cfg)
		if err == nil {
			pod, err := clientset.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
			if err == nil {
				return pod.Status.HostIP
			}
			cause = err.Error()
		} else {
			cause = err.Error()
		}
	}

	name, err := os.Hostname()
	if err != nil {
		fmt.Printf("Error: failed to get hostname from OS: %v\n", err)
	}

	fmt.Printf("Warning: hostname is taken from OS, this may be incorrect when running inside a container/pod: %s\n", cause)
	return name
}

// GetNamespace trying to read the namespace from the input
// if not set, it tries to read from an environment variable, and if not set there either
// it returns "default"
func GetNamespace(ns string) string {
	if ns != "" {
		return ns
	}
	if envNs := os.Getenv("POD_NAMESPACE"); envNs != "" {
		return envNs
	}
	return "default"
}
