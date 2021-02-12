package certificate

import (
	"fmt"
	"time"

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

	isAnnotatedResource := func(meta metav1.Object) bool {
		_, foundAnnotation := meta.GetAnnotations()[secretManagedAnnotatoinKey]
		return foundAnnotation
	}

	isWebhookConfig := func(meta metav1.Object) bool {
		return meta.GetName() == m.webhookName
	}

	// Watch only events for selected m.webhookName
	onEventForThisWebhook := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return isWebhookConfig(createEvent.Meta) || isAnnotatedResource(createEvent.Meta)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return isAnnotatedResource(deleteEvent.Meta)
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return isWebhookConfig(updateEvent.MetaOld) || isAnnotatedResource(updateEvent.MetaOld)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return isWebhookConfig(genericEvent.Meta) || isAnnotatedResource(genericEvent.Meta)
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
	reqLogger := m.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name).WithName("Reconcile")
	reqLogger.Info("Reconciling Certificates")

	elapsedToRotateCA := m.elapsedToRotateCAFromLastDeadline()
	elapsedToRotateServices := m.elapsedToRotateServicesFromLastDeadline()

	// Ensure that this Reconcile is not called after bad changes at
	// the certificate chain
	if elapsedToRotateCA > 0 {
		err := m.verifyTLS()
		if err != nil {
			reqLogger.Info(fmt.Sprintf("TLS certificate chain failed verification, forcing rotation, err: %v", err))
			// Force rotation
			elapsedToRotateCA = 0
		}
	}

	// We have pass expiration time for the CA
	if elapsedToRotateCA <= 0 {

		// If rotate fails runtime-controller manager will re-enqueue it, so
		// it will be retried
		err := m.rotateAll()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed rotating all certs")
		}

		// Re-calculate elapsedToRotate since we have generated new
		// certificates
		m.nextRotationDeadlineForCA()
		elapsedToRotateCA = m.elapsedToRotateCAFromLastDeadline()

		// Also recalculate it for serices certificate since they has changed
		m.nextRotationDeadlineForServices()
		elapsedToRotateServices = m.elapsedToRotateServicesFromLastDeadline()

	} else if elapsedToRotateServices <= 0 {
		// CA is ok but expiration but we have passed expiration time for service certificates
		err := m.rotateServicesWithOverlap()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed rotating services certs")
		}

		// Re-calculate elapsedToRotateServices since we have generated new
		// services certificates
		m.nextRotationDeadlineForServices()
		elapsedToRotateServices = m.elapsedToRotateServicesFromLastDeadline()
	}

	elapsedForCABundleCleanup, err := m.earliestElapsedForCACertsCleanup()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting ca bundle cleanup deadline")
	}

	// We have pass cleanup deadline let's do the cleanup
	if elapsedForCABundleCleanup <= 0 {
		err = m.cleanUpCABundle()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed cleaning up CABundle")
		}

		// Re-calculate cleanup deadline since we may have to remove some certs there
		elapsedForCABundleCleanup, err = m.earliestElapsedForCACertsCleanup()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed re-calculating ca bundle cleanup deadline")
		}
	}

	elapsedForServiceCertsCleanup, err := m.earliestElapsedForServiceCertsCleanup()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting service certs cleanup deadline")
	}

	// We have pass cleanup deadline let's do the cleanup
	if elapsedForServiceCertsCleanup <= 0 {
		err = m.cleanUpServiceCerts()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed cleaning up service certs")
		}

		// Re-calculate cleanup deadline since we may have to remove some certs there
		elapsedForServiceCertsCleanup, err = m.earliestElapsedForServiceCertsCleanup()
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed re-calculating service certs cleanup deadline")
		}
	}

	// Return the event that is going to happend sonner all services certificates rotation,
	// services certificate rotation or ca bundle cleanup
	m.log.Info("Calculating RequeueAfter", "elapsedToRotateCA", elapsedToRotateCA, "elapsedToRotateServices", elapsedToRotateServices, "elapsedForCABundleCleanup", elapsedForCABundleCleanup, "elapsedForServiceCertsCleanup", elapsedForServiceCertsCleanup)
	requeueAfter := min(elapsedToRotateCA, elapsedToRotateServices, elapsedForCABundleCleanup, elapsedForServiceCertsCleanup)

	m.log.Info(fmt.Sprintf("Certificates will be Reconcile on %s", m.now().Add(requeueAfter)))
	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func min(values ...time.Duration) time.Duration {
	m := time.Duration(0)
	for i, e := range values {
		if i == 0 || e < m {
			m = e
		}
	}
	return m
}
