package nmstatectl

import (
	"os"
	"fmt"

	restclient "k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"github.com/nmstate/k8s-node-net-conf/pkg/apis/nmstate.io/v1"
)

// ValidateNodeName check if the current host is a k8s node 
func ValidateNodeName(cfg *restclient.Config, nodeName string) bool {
	clientset, err := k8sclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building k8s client: %v\n", err)
	
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error listing all nodes: %v\n", err)
	}

	for _, node := range nodes.Items {
		if nodeName == node.GetName() {
			return true
		}
	}
	return false
}

// IsStateApplicable check if the state should be applied to current node
func IsStateApplicable(cfg *restclient.Config, state *v1.NodeNetworkState) bool {
	// TODO: we need a better way of finding out the node, than comparing to hostname
	// when running inside a pod, this should be simpler, by taking the node name from the pod's parameters
	nodeName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("Failed to get hostname: %v\n", err)
	}

	// node name validation is optional
	if cfg != nil && !ValidateNodeName(cfg, nodeName) {
		fmt.Printf("Warning: hostname '%s' was not found to be a valid node name\n", nodeName)
	}

	return nodeName == state.Spec.NodeName
}