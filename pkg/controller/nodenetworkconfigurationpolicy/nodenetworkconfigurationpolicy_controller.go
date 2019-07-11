package nodenetworkconfigurationpolicy

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	log = logf.Log.WithName("controller_nodenetworkconfigurationpolicy")
)

// Add creates a new NodeNetworkConfigurationPolicy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeNetworkConfigurationPolicy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func getNodeName() (string, error) {
	nodeName := os.Getenv("NODE_NAME")
	if len(nodeName) == 0 {
		return nodeName, fmt.Errorf("no NODE_NAME environment variable")
	}
	return nodeName, nil
}

func matches(nodeSelector map[string]string, labels map[string]string) bool {
	for key, value := range nodeSelector {
		if foundValue, hasKey := labels[key]; !hasKey || foundValue != value {
			return false
		}
	}
	return true
}

func nodeSelectorMatchesThisNode(cl client.Client, eventObject runtime.Object) bool {
	nodeName, err := getNodeName()
	if err != nil {
		log.Info("NODE_NAME not found for pod")
		return false
	}
	node := corev1.Node{}
	err = cl.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	if err != nil {
		log.Info("Cannot find corev1.Node", "nodeName", nodeName)
		return false
	}

	policyNodeSelector := eventObject.(*nmstatev1alpha1.NodeNetworkConfigurationPolicy).Spec.NodeSelector
	return matches(policyNodeSelector, node.ObjectMeta.Labels)
}

func forThisNodePredicate(cl client.Client) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return nodeSelectorMatchesThisNode(cl, createEvent.Object)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return nodeSelectorMatchesThisNode(cl, deleteEvent.Object)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return nodeSelectorMatchesThisNode(cl, updateEvent.ObjectOld) &&
				nodeSelectorMatchesThisNode(cl, updateEvent.ObjectNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return nodeSelectorMatchesThisNode(cl, genericEvent.Object)
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodenetworkconfigurationpolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NodeNetworkConfigurationPolicy
	err = c.Watch(&source.Kind{Type: &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}}, &handler.EnqueueRequestForObject{}, forThisNodePredicate(mgr.GetClient()))
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNodeNetworkConfigurationPolicy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeNetworkConfigurationPolicy{}

// ReconcileNodeNetworkConfigurationPolicy reconciles a NodeNetworkConfigurationPolicy object
type ReconcileNodeNetworkConfigurationPolicy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationPolicy object and makes changes based on the state read
// and what is in the NodeNetworkConfigurationPolicy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeNetworkConfigurationPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeNetworkConfigurationPolicy")

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	nodeNetworkState := &nmstatev1alpha1.NodeNetworkState{}
	nodeName, err := getNodeName()
	if err != nil {
		return reconcile.Result{}, err
	}
	nodeNetworkStateKey := types.NamespacedName{Name: nodeName}
	err = r.client.Get(context.TODO(), nodeNetworkStateKey, nodeNetworkState)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info(fmt.Sprintf("the NodeNetworkState for %s is not there yet, let's requeue", nodeName))
			// If there is no NodeNetworkState let's requeue could be that
			// we are in the middle of the creation
			return reconcile.Result{Requeue: true}, nil
		}
		// Error reading the nodeNetworkState - requeue the request.
		return reconcile.Result{}, err
	}

	// FIXME: We have to merge it somehow in case of multiple policies applied
	nodeNetworkState.Spec.DesiredState = instance.Spec.DesiredState
	err = r.client.Update(context.TODO(), nodeNetworkState)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
