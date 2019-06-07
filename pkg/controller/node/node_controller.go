package node

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	k8sHandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/handler"
)

var log = logf.Log.WithName("controller_node")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNode{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("node-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	//TODO: We only one creates/deletes and the timeouts
	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &k8sHandler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNode{}

// ReconcileNode reconciles a Node object
type ReconcileNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Node")

	// Fetch the Node instance
	instance := &corev1.Node{}
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

	//TODO: Manage deletes
	nodeNetworkStateKey := types.NamespacedName{
		Namespace: "default",
		Name:      request.Name,
	}
	// Create NodeNetworkState for this node
	nodeNetworkState := &nmstatev1.NodeNetworkState{}
	err = r.client.Get(context.TODO(), nodeNetworkStateKey, nodeNetworkState)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, fmt.Errorf("Erro accessing NodeNetworkState: %v", err)
		} else {
			nodeNetworkState.ObjectMeta = metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: "default",
			}
			nodeNetworkState.Spec = nmstatev1.NodeNetworkStateSpec{
				NodeName: request.Name,
			}
			// There is no NodeNetworkState for this node let's create it
			err = r.client.Create(context.TODO(), nodeNetworkState)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("Erro creating NodeNetworkState: %v", err)
			}
		}
	}

	handler, err := handler.New(r.client, request.Name)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Error finding nmstate-handler pod: %v", err)
	}

	currentState, err := handler.Nmstatectl("show")
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Error running nmstatectl show: %v", err)
	}

	// Let's update status with current network config from nmstatectl
	nodeNetworkState.Status = nmstatev1.NodeNetworkStateStatus{
		CurrentState: nmstatev1.State(currentState),
	}
	err = r.client.Status().Update(context.TODO(), nodeNetworkState)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Error Updateing status of NodeNetworkState: %v", err)
	}

	//TODO: Set a timer to refresh Status
	return reconcile.Result{}, nil
}
