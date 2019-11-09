package nodenetworkstate

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	k8shandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
)

var (
	log                     = logf.Log.WithName("controller_nodenetworkstate")
	nodenetworkstateRefresh time.Duration
)

func init() {
	refreshTime, isSet := os.LookupEnv("NODE_NETWORK_STATE_REFRESH_INTERVAL")
	if !isSet {
		panic("NODE_NETWORK_STATE_REFRESH_INTERVAL is mandatory")
	}
	intRefreshTime, err := strconv.Atoi(refreshTime)
	if err != nil {
		panic(fmt.Sprintf("Failed while converting evnironment variable to int: %v", err))
	}
	nodenetworkstateRefresh = time.Duration(intRefreshTime) * time.Second
}

// Add creates a new NodeNetworkState Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeNetworkState{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodenetworkstate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	forThisNode := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return nmstate.EventIsForThisNode(createEvent.Meta)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			// This controller responsability is updates, receiving
			// deletes is of no use
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return nmstate.EventIsForThisNode(updateEvent.MetaNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return nmstate.EventIsForThisNode(genericEvent.Meta)
		},
	}
	// Watch for changes to primary resource NodeNetworkState
	err = c.Watch(&source.Kind{Type: &nmstatev1alpha1.NodeNetworkState{}}, &k8shandler.EnqueueRequestForObject{}, forThisNode)
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileNodeNetworkState implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeNetworkState{}

// ReconcileNodeNetworkState reconciles a NodeNetworkState object
type ReconcileNodeNetworkState struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeNetworkState object and makes changes based on the state read
// and what is in the NodeNetworkState.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeNetworkState) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(1).Info("Reconciling NodeNetworkState")
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the NodeNetworkState instance
		instance := &nmstatev1alpha1.NodeNetworkState{}
		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		if err != nil {
			return err
		}

		err = nmstate.UpdateCurrentState(r.client, instance)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: nodenetworkstateRefresh}, nil
}
