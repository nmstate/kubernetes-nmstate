package utils

import (
	"fmt"
	"os"

	"github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

// ValidateNodeName check if the current host is a k8s node
func ValidateNodeName(kubeClient k8sclient.Interface, nodeName string) bool {
	nodes, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
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
func IsStateApplicable(kubeClient k8sclient.Interface, state *v1.NodeNetworkState, nodeName string) bool {
	if nodeName == state.Spec.NodeName {
		// node name validation is optional
		if !ValidateNodeName(kubeClient, nodeName) {
			fmt.Printf("Warning: hostname '%s' was not found to be a valid node name\n", nodeName)
		}

		if state.Name != state.Spec.NodeName {
			fmt.Printf("Warning: resource name '%s' does not match hostname '%s'\n", state.Name, nodeName)
		}
		return true
	}

	return false
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
