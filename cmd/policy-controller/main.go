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
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	clientset "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	informers "github.com/nmstate/kubernetes-nmstate/pkg/client/informers/externalversions"
	"github.com/nmstate/kubernetes-nmstate/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	nmstateClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building nmstate clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	nmstateInformerFactory := informers.NewSharedInformerFactory(nmstateClient, time.Second*30)

	controller := NewController(kubeClient, nmstateClient,
		nmstateInformerFactory.Nmstate().V1().NodeNetConfPolicies(),
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	nmstateInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
