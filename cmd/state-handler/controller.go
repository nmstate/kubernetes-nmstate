package main

import (
	"fmt"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	clientset "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	nmstatescheme "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned/scheme"
	informers "github.com/nmstate/kubernetes-nmstate/pkg/client/informers/externalversions/nmstate.io/v1"
	listers "github.com/nmstate/kubernetes-nmstate/pkg/client/listers/nmstate.io/v1"
)

const controllerAgentName = "state-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a NodeNetworkState is synced
	SuccessSynced = "Synced"
	// MessageResourceSynced is the message used for an Event fired when a NodeNetworkState
	// is synced successfully
	MessageResourceSynced = "NodeNetworkState synced successfully"
)

// Controller is the controller implementation for NodeNetworkState resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// nmstateclientset is a clientset for our own API group
	nmstateclientset clientset.Interface

	stateLister listers.NodeNetworkStateLister
	stateSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
	// each controller is associated with a specific host
	hostName string
	// namespace for CRDs of controller
	namespace string
	// used to make sure that we dont go into an update loop
	currentResourceVersion string
}

// NewController returns a new state controller
func NewController(
	kubeclientset kubernetes.Interface,
	nmstateclientset clientset.Interface,
	stateInformer informers.NodeNetworkStateInformer,
	hostName string,
	namespace string) *Controller {

	// Create event broadcaster
	// Add state-controller types to the default Kubernetes Scheme so Events can be
	// logged for state-controller types.
	nmstatescheme.AddToScheme(scheme.Scheme)
	klog.V(4).Info("Creating event broadcaster")
	fmt.Println("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		nmstateclientset: nmstateclientset,
		stateLister:      stateInformer.Lister(),
		stateSynced:      stateInformer.Informer().HasSynced,
		// TODO: take rate limiter parameters from conf
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeNetworkStates"),
		recorder:  recorder,
		hostName:  hostName,
		namespace: namespace,
	}

	klog.Info("Setting up event handlers")
	fmt.Println("Setting up event handlers")
	// Set up an event handler for when NodeNetworkState resources change
	stateInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodeNetworkState,
		UpdateFunc: func(old, new interface{}) {
			// TODO: support real update
			controller.enqueueNodeNetworkState(new)
		},
		// TODO: support CRD deletion
		DeleteFunc: controller.enqueueNodeNetworkState,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting NodeNetworkState controller")
	fmt.Println("Starting NodeNetworkState controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	fmt.Println("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.stateSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// Check if a resource with this namespace/name exists
	state, err := c.stateLister.NodeNetworkStates(c.namespace).Get(c.hostName)
	if err != nil {
		// The NodeNetworkState resource may not exist, in this case we create one
		if errors.IsNotFound(err) {
			state, err = nmstatectl.CreateResource(c.nmstateclientset.NmstateV1(), c.hostName, c.namespace)
			if err != nil {
				c.currentResourceVersion = ""
				return fmt.Errorf("failed to create state for node '%s': %v", c.hostName, err)
			}
			c.currentResourceVersion = state.ResourceVersion
			c.recorder.Event(state, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
		} else {
			return fmt.Errorf("failed to check if state exist for node '%s': %v", c.hostName, err)
		}
	}

	klog.Info("Starting workers")
	fmt.Println("Starting workers")
	// Launch two workers to process NodeNetworkState resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	fmt.Println("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")
	fmt.Println("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
		// TODO: worker should also poll current state to notify changes
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// NodeNetworkState resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			// this mechanism will implement retries in case that the call to nmstatectl
			// partially or completely failed
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		fmt.Printf("Successfully synced '%s'\n", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the NodeNetworkState resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the resource with this namespace/name
	state, err := c.stateLister.NodeNetworkStates(namespace).Get(name)
	if err != nil {
		// The NodeNetworkState resource may no longer exist, in which case we stop
		// processing.
		c.currentResourceVersion = ""
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("state '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Dont process in case this is our own update
	if state.ResourceVersion == c.currentResourceVersion {
		fmt.Printf("Incoming event was generated by controller (version '%s') - nothing to do\n", c.currentResourceVersion)
		return nil
	}

	// Do the actual handling of the state CRD
	state, err = nmstatectl.HandleResource(state, c.nmstateclientset.NmstateV1())
	if err != nil {
		c.currentResourceVersion = ""
		return err
	} else if state == nil {
		// node not managed, do nothing
		return nil
	}

	c.currentResourceVersion = state.ResourceVersion
	c.recorder.Event(state, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	return nil
}

// enqueueNodeNetworkState takes a NodeNetworkState resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than NodeNetworkState.
func (c *Controller) enqueueNodeNetworkState(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return
	}
	if name == c.hostName {
		fmt.Printf("event: %s was added to queue\n", key)
		c.workqueue.AddRateLimited(key)
	}
}
