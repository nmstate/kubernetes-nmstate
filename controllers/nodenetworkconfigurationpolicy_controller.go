package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
	"github.com/nmstate/kubernetes-nmstate/pkg/policyconditions"
	"github.com/nmstate/kubernetes-nmstate/pkg/selectors"
	"k8s.io/apimachinery/pkg/types"
)

var (
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

// NodeNetworkConfigurationPolicyReconciler reconciles a NodeNetworkConfigurationPolicy object
type NodeNetworkConfigurationPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NodeNetworkConfigurationPolicyReconciler) waitEnactmentCreated(enactmentKey types.NamespacedName) error {
	var enactment nmstatev1beta1.NodeNetworkConfigurationEnactment
	pollErr := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		err := r.Client.Get(context.TODO(), enactmentKey, &enactment)
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

func (r *NodeNetworkConfigurationPolicyReconciler) initializeEnactment(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) error {
	enactmentKey := nmstateapi.EnactmentKey(nodeName, policy.Name)
	log := r.Log.WithName("initializeEnactment").WithValues("policy", policy.Name, "enactment", enactmentKey.Name)
	// Return if it's already initialize or we cannot retrieve it
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	err := r.Client.Get(context.TODO(), enactmentKey, &enactment)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "failed getting enactment ")
	}
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("creating enactment")
		enactment = nmstatev1beta1.NewEnactment(nodeName, policy)
		err = r.Client.Create(context.TODO(), &enactment)
		if err != nil {
			return errors.Wrapf(err, "error creating NodeNetworkConfigurationEnactment: %+v", enactment)
		}
		err = r.waitEnactmentCreated(enactmentKey)
		if err != nil {
			return errors.Wrapf(err, "error waitting for NodeNetworkConfigurationEnactment: %+v", enactment)
		}
	} else {
		enactmentConditions := enactmentconditions.New(r.Client, enactmentKey)
		enactmentConditions.Reset()
	}

	return enactmentstatus.Update(r.Client, enactmentKey, func(status *nmstateapi.NodeNetworkConfigurationEnactmentStatus) {
		status.DesiredState = policy.Spec.DesiredState
		status.PolicyGeneration = policy.Generation
	})
}

// Reconcile reads that state of the cluster for a NodeNetworkConfigurationPolicy object and makes changes based on the state read
// and what is in the NodeNetworkConfigurationPolicy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeNetworkConfigurationPolicyReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("nodenetworkconfigurationpolicy", request.NamespacedName)

	// Fetch the NodeNetworkConfigurationPolicy instance
	instance := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error retrieving policy")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	policyconditions.Reset(r.Client, request.NamespacedName)

	err = r.initializeEnactment(*instance)
	if err != nil {
		log.Error(err, "Error initializing enactment")
	}

	enactmentConditions := enactmentconditions.New(r.Client, nmstateapi.EnactmentKey(nodeName, instance.Name))

	// Policy conditions will be updated at the end so updating it
	// does not impact at applying state, it will increase just
	// reconcile time.
	defer policyconditions.Update(r.Client, request.NamespacedName)

	policySelectors := selectors.NewFromPolicy(r.Client, *instance)
	unmatchingNodeLabels, err := policySelectors.UnmatchedNodeLabels(nodeName)
	if err != nil {
		log.Error(err, "failed checking node selectors")
		enactmentConditions.NotifyNodeSelectorFailure(err)
	}
	if len(unmatchingNodeLabels) > 0 {
		log.Info("Policy node selectors does not match node")
		enactmentConditions.NotifyNodeSelectorNotMatching(unmatchingNodeLabels)
		return ctrl.Result{}, nil
	}

	enactmentConditions.NotifyMatching()

	enactmentConditions.NotifyProgressing()
	nmstateOutput, err := nmstate.ApplyDesiredState(r.Client, instance.Spec.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy at desired state apply: %s, %v", nmstateOutput, err)

		enactmentConditions.NotifyFailedToConfigure(errmsg)
		log.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))
		return ctrl.Result{}, nil
	}
	log.Info("nmstate", "output", nmstateOutput)

	enactmentConditions.NotifySuccess()

	return ctrl.Result{}, nil
}

func (r *NodeNetworkConfigurationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1beta1.NodeNetworkConfigurationPolicy{}).
		Complete(r)
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
