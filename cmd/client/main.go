package main

import (
	"flag"
	"fmt"
	"os"

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
	// TODO: runmtime exception: "flag redefined: log_dir"
	//klog.InitFlags(nil)
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
			klog.Fatalf("Error listing all net conf policies (in %s): %v\n", *namespace, err)
		}

		for _, policy := range list.Items {
			fmt.Printf("Node net conf policy: %v\n", policy)
			// TODO: invoke policy handling
		}
	} else if *crdType == "state" {
		list, err := nmstateClient.NmstateV1().NodeNetworkStates(*namespace).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatalf("Error listing all net conf policies (in %s): %v\n", *namespace, err)
		}

		// TODO: we need a better way of finding out the node, than comparing to hostname
		// when running inside a pod, this should be simpler, by taking the node name from the pod's parameters
		nodeName, err := os.Hostname()
		if err != nil {
			klog.Fatalf("Failed to get hostname: %v\n", err)
		}

		if !nmstatectl.ValidateNodeName(cfg, *namespace, nodeName) {
			fmt.Printf("Warning: hostname '%s' was not found to be a valid node name\n", nodeName)
		}

		nodeFound := false
		for _, state := range list.Items {
			if nodeName == state.Spec.NodeName {
				nodeFound = true
				if state.Spec.Managed {
					if err = nmstatectl.Set(&state.Spec.DesiredState); err != nil {
						fmt.Printf("Failed set state on node: %v\n", err)
					}
				} else {
					fmt.Printf("Node '%s' is unmanaged by state '%s'\n", nodeName, state.Name)
				}

				// TODO: should we update current state for unmanaged nodes?
				if err = nmstatectl.Show(&state.Status.CurrentState); err != nil {
					fmt.Printf("Failed to fetch current state: %v\n", err)
				} else {
					if _, err := nmstateClient.NmstateV1().NodeNetworkStates(*namespace).Update(&state); err != nil {
						fmt.Printf("Failed to update state: %v\n", err)
					} else {
						fmt.Printf("Successfully update state '%s' on node '%s'\n", state.Name, nodeName)
					}
				}
			}
		}
		if !nodeFound {
			fmt.Printf("Warning: could not find state which apply to '%s'\n", nodeName)
		}
	} else {
		klog.Fatalf("Unknown CRD type to fetch: %s\n", *crdType)
	}
	klog.Flush()
}
