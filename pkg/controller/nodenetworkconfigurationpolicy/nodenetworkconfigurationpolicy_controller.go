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

	yaml "sigs.k8s.io/yaml"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
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

// It will return the policies with same labels at the one from argument
func (r *ReconcileNodeNetworkConfigurationPolicy) filterByNodeLabels(node corev1.Node) (nmstatev1alpha1.NodeNetworkConfigurationPolicyList, error) {
	policyList := nmstatev1alpha1.NodeNetworkConfigurationPolicyList{}
	filteredPolicyList := nmstatev1alpha1.NodeNetworkConfigurationPolicyList{}

	// Retrieve all the policies
	err := r.client.List(context.TODO(), &client.ListOptions{}, &policyList)
	if err != nil {
		return policyList, err
	}

	// Get the ones that fit this node
	for _, policy := range policyList.Items {
		if nodeSelectorMatchesThisNode(r.client, &policy) {
			filteredPolicyList.Items = append(filteredPolicyList.Items, policy)
		}
	}

	return filteredPolicyList, nil
}

func unmarshalInterfaces(state nmstatev1alpha1.State) (map[string]interface{}, []interface{}, error) {
	// Unmarshall interfaces state into unstructured golang
	var unstructuredState map[string]interface{}
	err := yaml.Unmarshal(state, &unstructuredState)
	if err != nil {
		return unstructuredState, []interface{}{}, fmt.Errorf("error unmarshaling state: %v", err)
	}
	return unstructuredState, unstructuredState["interfaces"].([]interface{}), nil
}

func mapByName(interfacesList []interface{}) (map[string]interface{}, error) {
	interfacesMap := map[string]interface{}{}
	for _, iface := range interfacesList {
		// Cast generic type to a map so we can search 'name' field
		interfaceMap := iface.(map[string]interface{})
		interfaceName, hasName := interfaceMap["name"]
		if !hasName {
			return interfaceMap, fmt.Errorf("no 'name' field at interface")
		}

		// Store in the map by 'name' so we can search for it
		interfacesMap[interfaceName.(string)] = interfaceMap
	}
	return interfacesMap, nil
}

func intersectionKeys(lhs map[string]interface{}, rhs map[string]interface{}) []string {
	intersectionKeys := []string{}
	for key, _ := range lhs {
		if _, hasKey := rhs[key]; hasKey {
			intersectionKeys = append(intersectionKeys, key)
		}
	}
	return intersectionKeys
}

// It will merge "interfaces" if they are not comflicting (there is not changes from the same interface) in case
// of conflicting and error is returned and no combination is done.
func combineState(inputState nmstatev1alpha1.State, outputState nmstatev1alpha1.State) (nmstatev1alpha1.State, error) {
	// If the output state is empty we just need to return the input
	if len(outputState) == 0 {
		return inputState, nil
	}

	_, inputInterfaces, err := unmarshalInterfaces(inputState)
	if err != nil {
		return outputState, fmt.Errorf("error unmarshaling input state: %v", err)
	}
	outputUnstructuredState, outputInterfaces, err := unmarshalInterfaces(outputState)
	if err != nil {
		return outputState, fmt.Errorf("error unmarshaling output state: %v", err)
	}

	inputMap, err := mapByName(inputInterfaces)
	if err != nil {
		return outputState, fmt.Errorf("error converting input to map: %v", err)
	}
	outputMap, err := mapByName(outputInterfaces)
	if err != nil {
		return outputState, fmt.Errorf("error converting output to map: %v", err)
	}

	// If we have a network interface at both input and output
	// don't do any combination
	intersectionKeys := intersectionKeys(inputMap, outputMap)
	if len(intersectionKeys) > 0 {
		return outputState, nil
	}
	// Add new configured interfaces to NodeNetworkState DesiredState
	outputInterfaces = append(outputInterfaces, inputInterfaces...)
	outputUnstructuredState["interfaces"] = outputInterfaces

	// Marshal back DesiredState to NodeNetworkState
	outputState, err = yaml.Marshal(outputUnstructuredState)
	if err != nil {
		return outputState, fmt.Errorf("error marshaling modified desired state: %v", err)
	}
	return outputState, nil
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

	nodeNetworkState := &nmstatev1alpha1.NodeNetworkState{}
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

	node := corev1.Node{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot find corev1.Node nodeName %s: %v", nodeName, err)
	}

	// It's going to also return reconciling instance but that's not an issue
	policyList, err := r.filterByNodeLabels(node)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error filtering policies by label: %v", err)
	}

	var combinedState nmstatev1alpha1.State
	for _, policy := range policyList.Items {
		combinedState, err = combineState(policy.Spec.DesiredState, combinedState)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	nodeNetworkState.Spec.DesiredState = combinedState

	// TODO: Use Patch instaed of Update
	err = r.client.Update(context.TODO(), nodeNetworkState)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
