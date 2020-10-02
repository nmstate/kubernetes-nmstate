package node

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Added for test purposes
type NmstateUpdater func(client client.Client, node *corev1.Node, namespace client.ObjectKey) error

var (
	log            = logf.Log.WithName("controller_node")
	nodeRefresh    = 5 * time.Second
	nmstateUpdater = nmstate.CreateOrUpdateNodeNetworkState
)

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("node-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// By default all this functors return true so controller watch all events,
	// but we only want to watch create/delete for current node.
	onCreationForThisNode := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return nmstate.EventIsForThisNode(createEvent.Meta)
		},
		DeleteFunc: func(event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}
	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}, onCreationForThisNode)
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that NodeReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &NodeReconciler{}

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Log    logr.Logger
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(1).Info("Reconciling Node")

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
	err = nmstateUpdater(r.client, instance, request.NamespacedName)
	if err != nil {
		err = fmt.Errorf("error at node reconcile creating NodeNetworkState: %v", err)
	}
	return reconcile.Result{RequeueAfter: nodeRefresh}, err
}

func (r *NMStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
