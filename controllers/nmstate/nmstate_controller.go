package nmstate

import (
	"context"
	"os"
	"path/filepath"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/names"
	nmstaterenderer "github.com/nmstate/kubernetes-nmstate/pkg/render"
	"github.com/openshift/cluster-network-operator/pkg/apply"
	"github.com/openshift/cluster-network-operator/pkg/render"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

// Add creates a new NMState Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNMState{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nmstate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NMState
	err = c.Watch(&source.Kind{Type: &nmstatev1beta1.NMState{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNMState implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNMState{}

// ReconcileNMState reconciles a NMState object
type ReconcileNMState struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NMState object and makes changes based on the state read
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNMState) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NMState")

	// We won't create more than one kubernetes-nmstate handler
	if request.Name != names.NMStateResourceName {
		reqLogger.Info("Ignoring NMState.nmstate.io without default name")
		return reconcile.Result{}, nil
	}

	// Fetch the NMState instance
	instance := &nmstatev1beta1.NMState{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
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

func (r *ReconcileNMState) applyCRDs(instance *nmstatev1beta1.NMState) error {
	data := render.MakeRenderData()
	return r.renderAndApply(instance, data, "crds", false)
}

func (r *ReconcileNMState) applyNamespace(instance *nmstatev1beta1.NMState) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(instance, data, "namespace", false)
}

func (r *ReconcileNMState) applyRBAC(instance *nmstatev1beta1.NMState) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(instance, data, "rbac", true)
}

func (r *ReconcileNMState) applyHandler(instance *nmstatev1beta1.NMState) error {
	data := render.MakeRenderData()
	// Register ToYaml template method
	data.Funcs["toYaml"] = nmstaterenderer.ToYaml
	// Prepare defaults
	masterExistsNoScheduleToleration := corev1.Toleration{
		Key:      "node-role.kubernetes.io/master",
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	}
	amd64ArchOnMasterNodeSelector := map[string]string{
		"beta.kubernetes.io/arch":        "amd64",
		"node-role.kubernetes.io/master": "",
	}
	amd64AndCRNodeSelector := instance.Spec.NodeSelector
	if amd64AndCRNodeSelector == nil {
		amd64AndCRNodeSelector = map[string]string{}
	}
	amd64AndCRNodeSelector["beta.kubernetes.io/arch"] = "amd64"

	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	data.Data["WebhookNodeSelector"] = amd64ArchOnMasterNodeSelector
	data.Data["WebhookTolerations"] = []corev1.Toleration{masterExistsNoScheduleToleration}
	data.Data["WebhookAffinity"] = corev1.Affinity{}
	data.Data["HandlerNodeSelector"] = amd64AndCRNodeSelector
	data.Data["HandlerTolerations"] = []corev1.Toleration{masterExistsNoScheduleToleration}
	data.Data["HandlerAffinity"] = corev1.Affinity{}
	// TODO: This is just a place holder to make template renderer happy
	//       proper variable has to be read from env or CR
	data.Data["CARotateInterval"] = ""
	data.Data["CAOverlapInterval"] = ""
	data.Data["CertRotateInterval"] = ""
	return r.renderAndApply(instance, data, "handler", true)
}

func (r *ReconcileNMState) renderAndApply(instance *nmstatev1beta1.NMState, data render.RenderData, sourceDirectory string, setControllerReference bool) error {
	var err error
	objs := []*uns.Unstructured{}

	objs, err = render.RenderDir(filepath.Join(names.ManifestDir, "kubernetes-nmstate", sourceDirectory), &data)
	if err != nil {
		return errors.Wrapf(err, "failed to render kubernetes-nmstate %s", sourceDirectory)
	}

	// If no file found in directory - return error
	if len(objs) == 0 {
		return errors.New("No manifests rendered")
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
