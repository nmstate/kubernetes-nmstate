package main

import (
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	nmstateioclientset "github.com/nmstate/k8s-node-net-conf/pkg/client/clientset/versioned"
	"github.com/nmstate/k8s-node-net-conf/pkg/nmstatectl"
	"github.com/nmstate/k8s-node-net-conf/pkg/utils"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	master     = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	namespace  = flag.String("n", "", "The namespace where the CRDs are created. If left blank and running via pod, it will be taken from there.")
	crdType    = flag.String("type", "state", "state|policy. Whether client should handle 'state' or 'policy' CRDs.")
	hostname   = flag.String("host", "", "Name of the host on which to enforce and report state. If left blank and running via pod, it will be taken from there.")
)

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v\n", err)
	}

	nmstateClient, err := nmstateioclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building nmstate clientset: %v\n", err)
	}

	// get name space even if not set as commandline parameter
	ns := utils.GetNamespace(*namespace)

	if *crdType == "policy" {
		list, err := nmstateClient.NmstateV1().NodeNetConfPolicies(ns).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatalf("Error listing all node net conf policies (in %s): %v\n", ns, err)
		}

		for _, policy := range list.Items {
			fmt.Printf("Node net conf policy: %v\n", policy)
			// TODO: invoke policy handling
		}
	} else if *crdType == "state" {
		list, err := nmstateClient.NmstateV1().NodeNetworkStates(ns).List(metav1.ListOptions{})
		if err != nil {
			klog.Fatalf("Error listing all node network states (in %s): %v\n", ns, err)
		}

		nodeFound := false
		name := utils.GetHostName(*hostname, cfg, ns)
		if name == "" {
			klog.Fatalf("Failed to get host name\n")
		}

		for _, state := range list.Items {
			if utils.IsStateApplicable(cfg, &state, name) {
				nodeFound = true
				if _, err = nmstatectl.HandleResource(&state, nmstateClient.NmstateV1()); err != nil {
					klog.Fatalf("Failed to handle resource '%s': %v\n", state.Name, err)
				}
				break
			}
		}
		if !nodeFound {
			fmt.Printf("Could not find an existing state which apply to node, will create one\n")
			if _, err = nmstatectl.CreateResource(nmstateClient.NmstateV1(), name, ns); err != nil {
				klog.Fatalf("Failed to create resource for node '%s': %v\n", name, err)
			}
		}
	} else {
		klog.Fatalf("Unknown CRD type to fetch: %s\n", *crdType)
	}
	klog.Flush()
}
