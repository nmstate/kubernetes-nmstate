package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	clientset "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	informers "github.com/nmstate/kubernetes-nmstate/pkg/client/informers/externalversions/nmstate.io/v1"
	listers "github.com/nmstate/kubernetes-nmstate/pkg/client/listers/nmstate.io/v1"
)

// Controller is the controller implementation for NodeNetConfPolicy resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// nmstateclientset is a clientset for our own API group
	nmstateclientset clientset.Interface

	policyLister listers.NodeNetConfPolicyLister
	policySynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new state controller
func NewController(
	kubeclientset kubernetes.Interface,
	nmstateclientset clientset.Interface,
	policyInformer informers.NodeNetConfPolicyInformer) *Controller {
	return nil
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting NodeNetConfPolicy controller")

	// TODO: Wait for the caches to be synced before starting workers

	klog.Info("Starting workers")

	// TODO: Launch two workers to process NodeNetworkState resources

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}
