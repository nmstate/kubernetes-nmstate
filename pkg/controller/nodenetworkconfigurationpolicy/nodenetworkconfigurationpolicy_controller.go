package nodenetworkconfigurationpolicy

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
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
	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/helper"
)

var (
	log      = logf.Log.WithName("controller_nodenetworkconfigurationpolicy")
	nodeName string
)

func init() {
	var isSet = false
	nodeName, isSet = os.LookupEnv("NODE_NAME")
	if !isSet || len(nodeName) == 0 {
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

func matches(nodeSelector map[string]string, labels map[string]string) bool {
	for key, value := range nodeSelector {
		if foundValue, hasKey := labels[key]; !hasKey || foundValue != value {
			return false
		}
	}
	return true
}

func nodeSelectorMatchesThisNode(cl client.Client, eventObject runtime.Object) bool {
	node := corev1.Node{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
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
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			if !nodeSelectorMatchesThisNode(cl, updateEvent.ObjectNew) {
				return false
			}

			// As described [1] if we want to ignore reconcile of status update we have
			// to check generation since it does not change on status updates also force
			// reconcile if finalizers have changes
			// [1] https://blog.openshift.com/kubernetes-operators-best-practices/
			generationIsDifferent := updateEvent.MetaNew.GetGeneration() != updateEvent.MetaOld.GetGeneration()
			return generationIsDifferent
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

	nmstateOutput, err := nmstate.ApplyDesiredState(instance.Spec.DesiredState)
	if err != nil {
		errmsg := fmt.Errorf("error reconciling NodeNetworkConfigurationPolicy at desired state apply: %s, %v", nmstateOutput, err)

		retryErr := r.setCondition(false, errmsg.Error(), request.NamespacedName)
		if retryErr != nil {
			reqLogger.Error(retryErr, "Failing condition update failed while reporting error: %v", errmsg)
		}
		reqLogger.Error(errmsg, fmt.Sprintf("Rolling back network configuration, manual intervention needed: %s", nmstateOutput))
		return reconcile.Result{}, nil
	}
	reqLogger.Info("nmstate", "output", nmstateOutput)

	err = r.setCondition(true, "successfully reconciled", request.NamespacedName)
	if err != nil {
		reqLogger.Error(err, "Success condition update failed while reporting success: %v", err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNodeNetworkConfigurationPolicy) setCondition(
	available bool,
	message string,
	policyName types.NamespacedName,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1alpha1.NodeNetworkConfigurationPolicy{}
		err := r.client.Get(context.TODO(), policyName, instance)
		if err != nil {
			return err
		}

		if available {
			setConditionSuccess(&instance.Status.Enactments, message)
		} else {
			setConditionFailed(&instance.Status.Enactments, message)
		}

		err = r.client.Status().Update(context.TODO(), instance)
		return err
	})
}

func setConditionFailed(enactments *nmstatev1alpha1.EnactmentList, message string) {
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailedToConfigure,
		message,
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailedToConfigure,
		"",
	)
}

func setConditionSuccess(enactments *nmstatev1alpha1.EnactmentList, message string) {
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionAvailable,
		corev1.ConditionTrue,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		message,
	)
	enactments.SetCondition(
		nodeName,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionFailing,
		corev1.ConditionFalse,
		nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionSuccessfullyConfigured,
		"",
	)
}

func desiredState(object runtime.Object) (nmstatev1alpha1.State, error) {
	var state nmstatev1alpha1.State
	switch v := object.(type) {
	default:
		return nmstatev1alpha1.State{}, fmt.Errorf("unexpected type %T", v)
	case *nmstatev1alpha1.NodeNetworkConfigurationPolicy:
		state = v.Spec.DesiredState
	}
	return state, nil
}
