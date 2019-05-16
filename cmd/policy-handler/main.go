// This controller is listening on NodeNetConfPolicy CRD events.
// When it gets such an event, find if it applies to the host it is running on
// according to node affinity and toleration, and then find out the list interfaces on which the policy
// should be applied. Based on that it creates a NodeNetworkState CRD that should be handled by a
// NodeNetworkState handler (either client or controller)
// It also controlls the lifetime of the NodeNetworkState CRDs based on the lifetime management of the
// NodeNetConfPolicy which owns them

package main

import (
	"flag"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	informers "github.com/nmstate/kubernetes-nmstate/pkg/client/informers/externalversions"
	"github.com/nmstate/kubernetes-nmstate/pkg/signals"
	"github.com/nmstate/kubernetes-nmstate/pkg/utils"
)

var (
	executionType = flag.String("execution-type", "", "\"controller|client\" Whether controller actively handling state changes OR only one-shot client should be started.")
	kubeconfig    = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	master        = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	namespace     = flag.String("n", "", "The namespace where the CRDs are created. If left blank and running via pod, it will be taken from there.")
	hostname      = flag.String("host", "", "Name of the host on which to enforce and report state. If left blank and running via pod, it will be taken from there.")
)

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v\n", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	nmstateClient, err := nmstate.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building nmstate clientset: %v\n", err)
	}

	// get name space even if not set as commandline parameter
	namespaceName := utils.GetNamespace(*namespace)

	switch *executionType {
	case "":
		panic("execution-type must be specified")
	case "controller":
		controller(kubeClient, nmstateClient)
	case "client":
		client(nmstateClient, namespaceName)
	}
}

func controller(kubeClient kubernetes.Interface, nmstateClient nmstate.Interface) {
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	nmstateInformerFactory := informers.NewSharedInformerFactory(nmstateClient, time.Second*30)

	controller := NewController(kubeClient, nmstateClient,
		nmstateInformerFactory.Nmstate().V1().NodeNetConfPolicies(),
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	nmstateInformerFactory.Start(stopCh)

	if err := controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func client(nmstateClient nmstate.Interface, namespaceName string) {
	// get name space even if not set as commandline parameter
	list, err := nmstateClient.NmstateV1().NodeNetConfPolicies(namespaceName).List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error listing all node net conf policies (in %s): %v\n", namespaceName, err)
	}

	for _, policy := range list.Items {
		fmt.Printf("Node net conf policy: %v\n", policy)
		// TODO: invoke policy handling
	}
	klog.Flush()
}
