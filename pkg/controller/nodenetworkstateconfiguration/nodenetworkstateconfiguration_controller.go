package nodenetworkstateconfiguration

import (
	"context"
	"fmt"
	"reflect"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	log = logf.Log.WithName("controller_nodenetworkstateconfiguration")
)

// Add creates a new NodeNetworkState Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeNetworkStateConfiguration{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func desiredState(object runtime.Object) (nmstatev1alpha1.State, error) {
	var state nmstatev1alpha1.State
	switch v := object.(type) {
	default:
		return nmstatev1alpha1.State{}, fmt.Errorf("unexpected type %T", v)
	case *nmstatev1alpha1.NodeNetworkState:
		state = v.Spec.DesiredState
	}
	return state, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodenetworkstateconfiguration-controller", mgr, controller.Options{Reconciler: r})
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
			eventIsForThisNode := nmstate.EventIsForThisNode(updateEvent.MetaNew)

			// As described [1] if we want to ignore reconcile of status update we have
			// to check generation since it does not change on status updates also force
			// reconcile if finalizers have changes
			// [1] https://blog.openshift.com/kubernetes-operators-best-practices/
			generationIsDifferent := updateEvent.MetaNew.GetGeneration() != updateEvent.MetaOld.GetGeneration()
			finalizersAreDifferent := !reflect.DeepEqual(updateEvent.MetaNew.GetFinalizers(), updateEvent.MetaOld.GetFinalizers())

			// we only care about desiredState changes
			oldDesiredState, err := desiredState(updateEvent.ObjectOld)
			if err != nil {
				log.Error(err, "retrieving desiredState from ObjectOld")
				return false
			}
			newDesiredState, err := desiredState(updateEvent.ObjectNew)
			if err != nil {
				log.Error(err, "retrieving desiredState from ObjectNew")
				return false
			}
			desiredStateIsDifferent := !reflect.DeepEqual(oldDesiredState, newDesiredState)

			return eventIsForThisNode && (generationIsDifferent || finalizersAreDifferent || desiredStateIsDifferent)
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

// blank assignment to verify that ReconcileNodeNetworkStateConfiguration implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeNetworkStateConfiguration{}

// ReconcileNodeNetworkStateConfiguration reconciles a NodeNetworkState object
type ReconcileNodeNetworkStateConfiguration struct {
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
func (r *ReconcileNodeNetworkStateConfiguration) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeNetworkStateConfiguration")

	// Fetch the NodeNetworkState instance
	instance := &nmstatev1alpha1.NodeNetworkState{}
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

	nmstateOutput, err := nmstate.ApplyDesiredState(instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error reconciling nodenetworkstate configuration at desired state apply: %v", err)
	}
	reqLogger.Info("nmstate", "output", nmstateOutput)

	return reconcile.Result{}, nil
}
