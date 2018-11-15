package nmstatectl

import (
	restclient "k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// ValidateNodeName check if the current host is a k8s node 
func ValidateNodeName(cfg *restclient.Config, namespace string, stateNodeName string) bool {
	clientset, err := k8sclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building k8s client: %v\n", err)
	
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error listing all nodes: %v\n", err)
	}

	for _, node := range nodes.Items {
		if stateNodeName == node.GetName() {
			return true
		}
	}
	return false
}
