package node

import (
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})
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

	handlerPod, err := r.handlerPod(request.Name)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info(fmt.Sprintf("Handler pod: %s", handlerPod.Name))
	kubeConfig, err := config.GetConfig()
	if err != nil {
		return reconcile.Result{}, err
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return reconcile.Result{}, err
	}
	// From https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/exec_util.go#L59
	req := kubeClient.RESTClient().Post().
		Resource("pods").
		Name(handlerPod.Name).
		Namespace(handlerPod.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: handlerPod.Spec.Containers[0].Name,
		Command:   []string{"/bin/sh", "-c", "nmstatectl show"},
		Stdin:     false,
		Stderr:    true,
		Stdout:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(kubeConfig, "POST", req.URL())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Error at executor init %v", err)
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout.String(), stderr.String(), err)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileNode) handlerPod(nodeName string) (corev1.Pod, error) {
	// Fetch the handler pod at this node
	podList := corev1.PodList{}
	listOptions := &client.ListOptions{}
	// We have to look by label, since is the  only cached stuff
	// matching by field does not work
	listOptions.MatchingLabels(map[string]string{"nmstate.io": "nmstate-handler"})
	err := r.client.List(context.TODO(), listOptions, &podList)
	if err != nil {
		return corev1.Pod{}, err
	}
	if len(podList.Items) == 0 {
		return corev1.Pod{}, fmt.Errorf("No nmstate handlers at cluster")
	}
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			return pod, nil
		}
	}
	// Just select the first one in case that more exists
	return corev1.Pod{}, fmt.Errorf("No nmstate handlers at %", nodeName)
}
