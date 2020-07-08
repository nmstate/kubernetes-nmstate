package certificate

import (
	"fmt"

	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func (m *Manager) Add(mgr manager.Manager) error {
	return m.add(mgr, m)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func (m *Manager) add(mgr manager.Manager, r reconcile.Reconciler) error {
	logger := m.log.WithName("add")
	// Create a new controller
	c, err := controller.New("certificate-controller", mgr, controller.Options{Reconciler: m})
	if err != nil {
		return errors.Wrap(err, "failed instanciating certificate controller")
	}

	isWebhookConfigOrAnnotatedResource := func(meta metav1.Object) bool {
		_, foundAnnotation := meta.GetAnnotations()[secretManagedAnnotatoinKey]
		return meta.GetName() == m.webhookName || foundAnnotation
	}

	// Watch only events for selected m.webhookName
	onEventForThisWebhook := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return isWebhookConfigOrAnnotatedResource(createEvent.Meta)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return isWebhookConfigOrAnnotatedResource(deleteEvent.Meta)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return isWebhookConfigOrAnnotatedResource(updateEvent.MetaOld)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return isWebhookConfigOrAnnotatedResource(genericEvent.Meta)
		},
	}

	logger.Info("Starting to watch secrets")
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, onEventForThisWebhook)
	if err != nil {
		return errors.Wrap(err, "failed watching Secret")
	}

	logger.Info("Starting to watch validatingwebhookconfiguration")
	err = c.Watch(&source.Kind{Type: &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}}, &handler.EnqueueRequestForObject{}, onEventForThisWebhook)
	if err != nil {
		return errors.Wrap(err, "failed watching ValidatingWebhookConfiguration")
	}

	logger.Info("Starting to watch mutatingwebhookconfiguration")
	err = c.Watch(&source.Kind{Type: &admissionregistrationv1beta1.MutatingWebhookConfiguration{}}, &handler.EnqueueRequestForObject{}, onEventForThisWebhook)
	if err != nil {
		return errors.Wrap(err, "failed watching MutatingWebhookConfiguration")
	}

	return nil
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (m *Manager) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := m.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(1).Info("Reconciling Certificates")

	elapsedToRotate := m.nextElapsedToRotate()

	// We have pass expiration time or it was forced
	if elapsedToRotate <= 0 {

		// If rotate fails runtime-controller manager will re-enqueue it, so
		// it will be retried
		err := m.rotate()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed rotating certs")
		}

		// Re-calculate elapsedToRotate since we have generated new
		// certificates
		elapsedToRotate = m.nextElapsedToRotate()

	}

	m.log.Info(fmt.Sprintf("Certificates will be Reconcile on %s", m.now().Add(elapsedToRotate)))
	return reconcile.Result{RequeueAfter: elapsedToRotate}, nil
}
