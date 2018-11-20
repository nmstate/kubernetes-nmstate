package main

import (
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	nmstateioclientset "github.com/nmstate/k8s-node-net-conf/pkg/client/clientset/versioned"
	"github.com/nmstate/k8s-node-net-conf/pkg/nmstatectl"
)

var (
	kuberconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	master      = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	namespace   = flag.String("n", "default", "The namespace where the CRDs are created.")
	crdType     = flag.String("type", "state", "state|policy. Whether client should handle 'state' or 'policy' CRDs.")
)

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kuberconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v\n", err)
	}

	nmstateClient, err := nmstateioclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building nmstate clientset: %v\n", err)
	}

	if *crdType == "policy" {
		list, err := nmstateClient.NmstateV1().NodeNetConfPolicies(*namespace).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatalf("Error listing all node net conf policies (in %s): %v\n", *namespace, err)
		}

		for _, policy := range list.Items {
			fmt.Printf("Node net conf policy: %v\n", policy)
			// TODO: invoke policy handling
		}
	} else if *crdType == "state" {
		list, err := nmstateClient.NmstateV1().NodeNetworkStates(*namespace).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatalf("Error listing all node network states (in %s): %v\n", *namespace, err)
		}

		nodeFound := false
		name := nmstatectl.GetHostName()
		if name == "" {
			klog.Fatalf("Failed to get host name\n")
		}

		for _, state := range list.Items {
			if nmstatectl.IsStateApplicable(cfg, &state, name) {
				nodeFound = true
				if _, err = nmstatectl.HandleResource(&state, nmstateClient.NmstateV1()); err != nil {
					klog.Fatalf("Failed to handle resource '%s': %v\n", state.Name, err)
				}
				break
			}
		}
		if !nodeFound {
			fmt.Printf("Could not find an existing state which apply to node, will create one\n")
			if _, err = nmstatectl.CreateResource(nmstateClient.NmstateV1(), name, *namespace); err != nil {
				klog.Fatalf("Failed to create resource for node '%s': %v\n", name, err)
			}
		}
	} else {
		klog.Fatalf("Unknown CRD type to fetch: %s\n", *crdType)
	}
	klog.Flush()
}
