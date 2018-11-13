package main

import (
	"flag"
	"fmt"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	nmstateioclientset "github.com/nmstate/k8s-node-net-conf/pkg/client/clientset/versioned"
)

var (
	kuberconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	master      = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
)

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kuberconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}

	nmstateClient, err := nmstateioclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building nmstate clientset: %v", err)
	}

	list, err := nmstateClient.NmstateV1().NodeNetConfPolicies("default").List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error listing all net conf policies: %v", err)
	}

	for _, policy := range list.Items {
		fmt.Printf("node net conf policy: %v", policy)
	}
}
