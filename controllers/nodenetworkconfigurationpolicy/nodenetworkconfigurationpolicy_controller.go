package nodenetworkconfigurationpolicy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/enactmentstatus"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/enactmentstatus/conditions"
	"github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/policyconditions"
	"github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy/selectors"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
)

var (
	log            = logf.Log.WithName("controller_nodenetworkconfigurationpolicy")
	nodeName       string
	watchPredicate = predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			// [1] https://blog.openshift.com/kubernetes-operators-best-practices/
			generationIsDifferent := updateEvent.MetaNew.GetGeneration() != updateEvent.MetaOld.GetGeneration()
			return generationIsDifferent
		},
	}
)

func init() {
	if !environment.IsHandler() {
		return
	}

	nodeName = environment.NodeName()
	if len(nodeName) == 0 {
		panic("NODE_NAME is mandatory")
	}
}

// Add creates a new NodeNetworkConfigurationPolicy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeNetworkConfigurationPolicy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodenetworkconfigurationpolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NodeNetworkConfigurationPolicy
	err = c.Watch(&source.Kind{Type: &nmstatev1beta1.NodeNetworkConfigurationPolicy{}}, &handler.EnqueueRequestForObject{}, watchPredicate)
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

func (r *ReconcileNodeNetworkConfigurationPolicy) waitEnactmentCreated(enactmentKey types.NamespacedName) error {
	var enactment nmstatev1beta1.NodeNetworkConfigurationEnactment
	pollErr := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		err := r.client.Get(context.TODO(), enactmentKey, &enactment)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Let's retry after a while, sometimes it takes some time
				// for enactment to be created
				return false, nil
			}
			return false, err
		}
		return true, nil
	})

	return pollErr
}

func (r *ReconcileNodeNetworkConfigurationPolicy) initializeEnactment(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) error {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	logger := log.WithName("initializeEnactment").WithValues("policy", policy.Name, "enactment", enactmentKey.Name)
	// Return if it's already initialize or we cannot retrieve it
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.client.Get(context.TODO(), enactmentKey, &enactment)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed getting enactment ")
	}
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("creating enactment")
		enactment = nmstatev1beta1.NewEnactment(nodeName, policy)
		err = r.client.Create(context.TODO(), &enactment)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error creating NodeNetworkConfigurationEnactment: %+v", enactment))
		}
		err = r.waitEnactmentCreated(enactmentKey)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error waitting for NodeNetworkConfigurationEnactment: %+v", enactment))
		}
	} else {
		enactmentConditions := enactmentconditions.New(r.client, enactmentKey)
		enactmentConditions.Reset()
	}

	return enactmentstatus.Update(r.client, enactmentKey, func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
		status.DesiredState = policy.Spec.DesiredState
		status.PolicyGeneration = policy.Generation
	})
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationPolicy object and makes changes based on the state read
// and what is in the NodeNetworkConfigurationPolicy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeNetworkConfigurationPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeNetworkConfigurationPolicy")

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	policyconditions.Reset(r.client, request.NamespacedName)

	err = r.initializeEnactment(*instance)
	if err != nil {
		log.Error(err, "Error initializing enactment")
	}

	enactmentConditions := enactmentconditions.New(r.client, nmstateapi.EnactmentKey(nodeName, instance.Name))

	// Policy conditions will be updated at the end so updating it
	// does not impact at applying state, it will increase just
	// reconcile time.
	defer policyconditions.Update(r.client, request.NamespacedName)

	policySelectors := selectors.NewFromPolicy(r.client, *instance)
	unmatchingNodeLabels, err := policySelectors.UnmatchedNodeLabels(nodeName)
	if err != nil {
		reqLogger.Error(err, "failed checking node selectors")
		enactmentConditions.NotifyNodeSelectorFailure(err)
	}
	if len(unmatchingNodeLabels) > 0 {
		reqLogger.Info("Policy node selectors does not match node")
		enactmentConditions.NotifyNodeSelectorNotMatching(unmatchingNodeLabels)
		return reconcile.Result{}, nil
	}

	enactmentConditions.NotifyMatching()

	enactmentConditions.NotifyProgressing()
	nmstateOutput, err := nmstate.ApplyDesiredState(r.client, instance.Spec.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy at desired state apply: %s, %v", nmstateOutput, err)

		enactmentConditions.NotifyFailedToConfigure(errmsg)
		reqLogger.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))
		return reconcile.Result{}, nil
	}
	reqLogger.Info("nmstate", "output", nmstateOutput)

	enactmentConditions.NotifySuccess()

	return reconcile.Result{}, nil
}

func desiredState(object runtime.Object) (nmstateapi.State, error) {
	var state nmstateapi.State
	switch v := object.(type) {
	default:
		return nmstateapi.State{}, fmt.Errorf("unexpected type %T", v)
	case *nmstatev1beta1.NodeNetworkConfigurationPolicy:
		state = v.Spec.DesiredState
	}
	return state, nil
}
