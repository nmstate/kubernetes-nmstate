// This client could be a invoked to handle NodeNetworkState CRDs or NodeNetConfPolicy CRDs
//
// NodeNetworkState Mode
// Client reads all NodeNetworkState CRDs, find the one that apply to the host it is running on
// according to the NodeName field of thr CRD.
// For such CRD it uses nmstatectl to enforce the desired state
// and then to report current state in the NodeNetworkState CRD.
// Notes:
// (1) The client cannot handle CRD deletions - and will not do state cleanup upon deletion
// (2) The client can handle updates to the CRD in some cases, since nmstate will try to enforce the new state
//     however, for parameters which do not require full state (e.g. static IPs), updates will not be handled by the client
// For full CRD lifetime management, the controller should be used
//
// NodeNetConfPolicy Mode
// Client reads all NodeNetworkState CRDs, find which of them apply to the host it is running on
// according to node affinity and toleration, and then find out the list interfaces on which the policy
// should be applied. Based on that it creates a NodeNetworkState CRD that should be handled by a
// NodeNetworkState handler (either client or controller)

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

		nodeFound := false
		for _, state := range list.Items {
			if nmstatectl.IsStateApplicable(cfg, &state) {
				nodeFound = true
				if err = nmstatectl.HandleResource(&state, nmstateClient.NmstateV1()); err != nil {
					fmt.Printf("Failed to handle resource '%s': %v\n", state.Name, err)
				}
			}
		}
		if !nodeFound {
			fmt.Printf("Warning: could not find state which apply to node\n")
		}
	} else {
		klog.Fatalf("Unknown CRD type to fetch: %s\n", *crdType)
	}
	klog.Flush()
}
