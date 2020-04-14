package nmstate

import (
	"context"
	"os"
	"path/filepath"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
	"github.com/nmstate/kubernetes-nmstate/pkg/names"
	"github.com/openshift/cluster-network-operator/pkg/apply"
	"github.com/openshift/cluster-network-operator/pkg/render"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log = logf.Log.WithName("controller_nmstate")
)

// Add creates a new NMstate Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNMstate{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nmstate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NMstate
	err = c.Watch(&source.Kind{Type: &nmstatev1alpha1.NMstate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNMstate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNMstate{}

// ReconcileNMstate reconciles a NMstate object
type ReconcileNMstate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NMstate object and makes changes based on the state read
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNMstate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NMstate")

	// Fetch the NMstate instance
	instance := &nmstatev1alpha1.NMstate{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cle  anup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = r.applyCRDs(instance)
	if err != nil {
		errors.Wrap(err, "failed applying CRDs")
		return reconcile.Result{}, err
	}

	err = r.applyNamespace(instance)
	if err != nil {
		errors.Wrap(err, "failed applying Namespace")
		return reconcile.Result{}, err
	}

	err = r.applyRBAC(instance)
	if err != nil {
		errors.Wrap(err, "failed applying RBAC")
		return reconcile.Result{}, err
	}

	err = r.applyHandler(instance)
	if err != nil {
		errors.Wrap(err, "failed applying Handler")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Reconcile complete.")
	return reconcile.Result{}, nil
}

func (r *ReconcileNMstate) applyCRDs(instance *nmstatev1alpha1.NMstate) error {
	data := render.MakeRenderData()
	return r.renderAndApply(instance, data, "crds", false)
}

func (r *ReconcileNMstate) applyNamespace(instance *nmstatev1alpha1.NMstate) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(instance, data, "namespace", false)
}

func (r *ReconcileNMstate) applyRBAC(instance *nmstatev1alpha1.NMstate) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(instance, data, "rbac", true)
}

func (r *ReconcileNMstate) applyHandler(instance *nmstatev1alpha1.NMstate) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(instance, data, "handler", true)
}

func (r *ReconcileNMstate) renderAndApply(instance *nmstatev1alpha1.NMstate, data render.RenderData, sourceDirectory string, setControllerReference bool) error {
	var err error
	objs := []*uns.Unstructured{}

	objs, err = render.RenderDir(filepath.Join(names.ManifestDir, "kubernetes-nmstate", sourceDirectory), &data)
	if err != nil {
		return errors.Wrapf(err, "failed to render kubernetes-nmstate %s", sourceDirectory)
	}

	for _, obj := range objs {
		// RenderDir seems to add an extra null entry to the list. It appears to be because of the
		// nested templates. This just makes sure we don't try to apply an empty obj.
		if obj.GetName() == "" {
			continue
		}
		if setControllerReference {
			// Set the controller refernce. When the CR is removed, it will remove the CRDs as well
			err = controllerutil.SetControllerReference(instance, obj, r.scheme)
			if err != nil {
				return errors.Wrap(err, "failed to set owner reference")
			}
		}

		// Now apply the object
		err = apply.ApplyObject(context.TODO(), r.client, obj)
		if err != nil {
			return errors.Wrapf(err, "failed to apply object %v", obj)
		}
	}
	return nil
}
