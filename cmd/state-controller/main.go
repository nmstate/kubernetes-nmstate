package main

import (
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	clientset "github.com/nmstate/k8s-node-net-conf/pkg/client/clientset/versioned"
	informers "github.com/nmstate/k8s-node-net-conf/pkg/client/informers/externalversions"
	"github.com/nmstate/k8s-node-net-conf/pkg/signals"
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

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
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

	ns := utils.GetNamespace(*namespace)
	controller := NewController(
		kubeClient,
		nmstateClient,
		nmstateInformerFactory.Nmstate().V1().NodeNetworkStates(),
		utils.GetHostName(*hostname, cfg, ns),
		ns,
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	nmstateInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
