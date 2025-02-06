/*
 * Copyright 2022 Kube Admission Webhook Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func (m *Manager) Add(mgr manager.Manager) error {
	return m.add(mgr)
}

type ctrlPredicate[T metav1.Object] struct {
	m *Manager
}

func (p ctrlPredicate[T]) Create(e event.TypedCreateEvent[T]) bool {
	return p.m.isWebhookConfig(e.Object) || (isAnnotatedResource(e.Object) && p.m.isGeneratedSecret(e.Object))
}

func (p ctrlPredicate[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	return isAnnotatedResource(e.Object) && p.m.isGeneratedSecret(e.Object)
}

func (p ctrlPredicate[T]) Update(e event.TypedUpdateEvent[T]) bool {
	return p.m.isWebhookConfig(e.ObjectOld) ||
		(isAnnotatedResource(e.ObjectOld) && p.m.isGeneratedSecret(e.ObjectOld))
}

func (p ctrlPredicate[T]) Generic(e event.TypedGenericEvent[T]) bool {
	return p.m.isWebhookConfig(e.Object) || (isAnnotatedResource(e.Object) && p.m.isGeneratedSecret(e.Object))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func (m *Manager) add(mgr manager.Manager) error {
	logger := m.log.WithName("add")
	// Create a new controller
	c, err := controller.New("certificate-controller", mgr, controller.Options{Reconciler: m})
	if err != nil {
		return errors.Wrap(err, "failed instanciating certificate controller")
	}

	logger.Info("Starting to watch secrets")
	if err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&corev1.Secret{},
			&handler.TypedEnqueueRequestForObject[*corev1.Secret]{},
			&ctrlPredicate[*corev1.Secret]{m: m},
		),
	); err != nil {
		return fmt.Errorf("unable to watch secrets: %w", err)
	}

	logger.Info("Starting to watch validatingwebhookconfiguration")
	if err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&admissionregistrationv1.ValidatingWebhookConfiguration{},
			&handler.TypedEnqueueRequestForObject[*admissionregistrationv1.ValidatingWebhookConfiguration]{},
			&ctrlPredicate[*admissionregistrationv1.ValidatingWebhookConfiguration]{m: m},
		),
	); err != nil {
		return errors.Wrap(err, "failed watching ValidatingWebhookConfiguration")
	}

	logger.Info("Starting to watch mutatingwebhookconfiguration")
	if err = c.Watch(
		source.Kind(
			mgr.GetCache(),
			&admissionregistrationv1.MutatingWebhookConfiguration{},
			&handler.TypedEnqueueRequestForObject[*admissionregistrationv1.MutatingWebhookConfiguration]{},
			&ctrlPredicate[*admissionregistrationv1.MutatingWebhookConfiguration]{m: m},
		),
	); err != nil {
		return errors.Wrap(err, "failed watching MutatingWebhookConfiguration")
	}

	return nil
}

func isAnnotatedResource(object metav1.Object) bool {
	_, foundAnnotation := object.GetAnnotations()[secretManagedAnnotatoinKey]
	return foundAnnotation
}

func (m *Manager) isWebhookConfig(object metav1.Object) bool {
	return object.GetName() == m.webhookName
}

func (m *Manager) isCASecret(object metav1.Object) bool {
	return object.GetName() == m.caSecretKey().Name
}

func (m *Manager) isServiceSecret(object metav1.Object) bool {
	webhookConf, err := m.readyWebhookConfiguration()
	if err != nil {
		m.log.Info(fmt.Sprintf("failed checking if it's a generated secret: failed getting webhook configuration: %v", err))
		return false
	}

	services, err := m.getServicesFromConfiguration(webhookConf)
	if err != nil {
		m.log.Info(fmt.Sprintf("failed checking if it's a generated secret: failed getting webhook configuration services: %v", err))
		return false
	}

	for service := range services {
		if object.GetName() == service.Name {
			return true
		}
	}
	return false
}

func (m *Manager) isGeneratedSecret(object metav1.Object) bool {
	return m.isCASecret(object) || m.isServiceSecret(object)
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (m *Manager) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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

	// Return the event that is going to happen sooner all services certificates rotation,
	// services certificate rotation or ca bundle cleanup
	m.log.Info("Calculating RequeueAfter", "elapsedToRotateCA", elapsedToRotateCA,
		"elapsedToRotateServices", elapsedToRotateServices, "elapsedForCABundleCleanup",
		elapsedForCABundleCleanup, "elapsedForServiceCertsCleanup", elapsedForServiceCertsCleanup)
	requeueAfter := minDuration(elapsedToRotateCA, elapsedToRotateServices, elapsedForCABundleCleanup, elapsedForServiceCertsCleanup)

	m.log.Info(fmt.Sprintf("Certificates will be Reconcile on %s", m.now().Add(requeueAfter)))
	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func minDuration(values ...time.Duration) time.Duration {
	m := time.Duration(0)
	for i, e := range values {
		if i == 0 || e < m {
			m = e
		}
	}
	return m
}
