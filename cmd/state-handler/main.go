package main

import (
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"
	"time"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	informers "github.com/nmstate/kubernetes-nmstate/pkg/client/informers/externalversions"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
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

	if hostname == nil || *hostname == "" {
		if envHostname, exists := os.LookupEnv("NODE_NAME"); !exists {
			klog.Fatalf("Failed to get host name: missing NODE_NAME env variable")
		} else {
			*hostname = envHostname
		}
	}

	switch *executionType {
	case "":
		panic("execution-type must be specified")
	case "controller":
		controller(kubeClient, nmstateClient, *hostname, namespaceName)
	case "client":
		client(kubeClient, nmstateClient, *hostname, namespaceName)
	}
}

func controller(kubeClient kubernetes.Interface, nmstateClient nmstate.Interface, hostName string, namespaceName string) {
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	nmstateInformerFactory := informers.NewSharedInformerFactory(nmstateClient, time.Second*30)

	controller := NewController(
		kubeClient,
		nmstateClient,
		nmstateInformerFactory.Nmstate().V1().NodeNetworkStates(),
		hostName,
		namespaceName,
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	nmstateInformerFactory.Start(stopCh)

	if err := controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func client(kubeClient kubernetes.Interface, nmstateClient nmstate.Interface, hostName string, namespaceName string) {
	list, err := nmstateClient.NmstateV1().NodeNetworkStates(namespaceName).List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error listing all node network states (in %s): %v\n", namespaceName, err)
	}

	nodeFound := false
	for _, state := range list.Items {
		if utils.IsStateApplicable(kubeClient, &state, hostName) {
			nodeFound = true
			if _, err = nmstatectl.HandleResource(&state, nmstateClient.NmstateV1()); err != nil {
				klog.Fatalf("Failed to handle resource '%s': %v\n", state.Name, err)
			}
			break
		}
	}
	if !nodeFound {
		fmt.Printf("Could not find an existing state which apply to node, will create one\n")
		if _, err = nmstatectl.CreateResource(nmstateClient.NmstateV1(), hostName, namespaceName); err != nil {
			klog.Fatalf("Failed to create resource for node '%s': %v\n", hostName, err)
		}
	}
	klog.Flush()
}
