package certificate

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate/triple"
)

// Manager do the CA and service certificate/key generation and expiration
// handling.
// It will generate one CA for the webhook configuration and a
// secret per Service referenced on it. One unique instance has to run at
// at cluster to monitor expiration time and do rotations.
type Manager struct {
	// client contains the controller-runtime client from the manager.
	client client.Client

	// webhookName The Mutating or Validating Webhook configuration name
	webhookName string

	// webhookType The Mutating or Validating Webhook configuration type
	webhookType WebhookType

	// caKeyPair contains the generated CA certificate and key
	caCert *x509.Certificate

	// now is an artifact to do some unit testing without waiting for
	// expiration time.
	now func() time.Time

	// certsDuration configurated duration for CA and service certificates
	// there is no distintion between the two to simplify manager logic
	// and monitor only CA certificate.
	certsDuration time.Duration

	// log initialized log that containes the webhook configuration name and
	// namespace so it's easy to debug.
	log logr.Logger
}

type WebhookType string

const (
	MutatingWebhook   WebhookType = "Mutating"
	ValidatingWebhook WebhookType = "Validating"
	OneYearDuration               = 365 * 24 * time.Hour
)

// NewManager with create a certManager that generated a secret per service
// at the webhook TLS http server.
// It will also starts at cert manager [1] that will update them if they expire.
// The generate certificate include the following fields:
// DNSNames (for every service the webhook refers too):
//	   - ${service.Name}
//	   - ${service.Name}.${service.namespace}
//	   - ${service.Name}.${service.namespace}.svc
// Subject:
// 	  - CN: ${webhookName}
// Usages:
//	   - UsageDigitalSignature
//	   - UsageKeyEncipherment
//	   - UsageServerAuth
//
// It will also update the webhook caBundle field with the cluster CA cert and
// approve the generated cert/key with k8s certification approval mechanism
func NewManager(
	client client.Client,
	webhookName string,
	webhookType WebhookType,
	certsDuration time.Duration,
) *Manager {

	m := &Manager{
		client:        client,
		webhookName:   webhookName,
		webhookType:   webhookType,
		now:           time.Now,
		certsDuration: certsDuration,
		log: logf.Log.WithName("webhook/server/certificate/manager").
			WithValues("webhookType", webhookType, "webhookName", webhookName),
	}
	return m
}

// Start the cert manager until stopCh is closed, the cert manager is in charge
// rotation of certificates if needed.
//
// It  implemenets Runnable [1] so manager can add this to a
// controller runtime manager
//
// [1] https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/manager/manager.go#L208
func (m *Manager) Start(stopCh <-chan struct{}) error {
	m.log.Info("Starting cert manager")

	m.loadCACertFromCABundle()

	wait.Until(func() {
		m.waitForDeadlineAndRotate()
	}, time.Second, stopCh)

	return nil
}

// In case the manager is restarted we have to load the current CA instead
// of generating new one
func (m *Manager) loadCACertFromCABundle() {
	caBundle, err := m.CABundle()
	if err != nil || len(caBundle) == 0 {
		return
	}

	cas, err := triple.ParseCertsPEM(caBundle)
	if err != nil || len(cas) == 0 {
		return
	}
	m.caCert = cas[0]
}

func (m *Manager) waitForDeadlineAndRotate() {
	deadline := m.nextRotationDeadline()
	now := m.now()
	elapsedToRotate := deadline.Sub(now)
	m.log.Info(fmt.Sprintf("Cert rotation times {now: %s, deadline: %s, elapsedToRotate: %s}", now, deadline, elapsedToRotate))
	if elapsedToRotate > 0 {
		m.log.Info(fmt.Sprintf("Waiting %v for next certificate rotation", elapsedToRotate))

		timer := time.NewTimer(elapsedToRotate)
		defer timer.Stop()

		select {
		case <-timer.C:
		}
	}
	// Retry rotate if it fails no timeout is added here since this is
	// the only thing that cert manager has to do, server will be function
	// until it reached expiricy in case of error somewhere.
	err := wait.PollImmediateInfinite(32*time.Second, m.rotateWaitCondition)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Unable to rotate certs: %v", err))
	}
}

// In case of running it under controller-runtime the manager has to be running
// at one pod per cluster since it generate new caBundle and it has to be unique
// per tls secrets if done otherwise caBundle get overwritten and it could not match
// the TLS secret.
func (m *Manager) NeedLeaderElection() bool {
	return true
}

// rotateWaitCondition wraps the rotate function into a `wait` Condition
// it will transform the error into a `not ready` flag and log and
// store error with `utilruntime.HandleError`.
func (m *Manager) rotateWaitCondition() (bool, error) {
	err := m.rotate()
	if err != nil {
		utilruntime.HandleError(err)
		return false, nil
	}
	return true, nil
}

func (m *Manager) rotate() error {

	m.log.Info("Rotating TLS cert/key")

	caKeyPair, err := triple.NewCA(m.webhookName, m.certsDuration)
	if err != nil {
		return errors.Wrap(err, "failed generating CA cert/key")
	}

	m.caCert = caKeyPair.Cert

	err = m.updateWebhookCABundle()
	if err != nil {
		return errors.Wrap(err, "failed to update CA bundle at webhook")
	}

	webhookConf, err := m.readyWebhookConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to reading configuration")
	}

	for _, clientConfig := range m.clientConfigList(webhookConf) {
		service := types.NamespacedName{Name: clientConfig.Service.Name, Namespace: clientConfig.Service.Namespace}
		keyPair, err := triple.NewServerKeyPair(
			caKeyPair,
			service.Name+"."+service.Namespace+".pod.cluster.local",
			service.Name,
			service.Namespace,
			"cluster.local",
			nil,
			nil,
			m.certsDuration,
		)
		if err != nil {
			return errors.Wrapf(err, "failed creating server key/cert for service %+v", service)
		}
		m.applyTLSSecret(service, keyPair)
	}

	return nil
}

// nextRotationDeadline returns a value for the threshold at which the
// current certificate should be rotated, 80%+/-10% of the expiration of the
// certificate.
func (m *Manager) nextRotationDeadline() time.Time {
	if m.caCert == nil {
		m.log.Info("Certificates not created, forcing roration")
		return m.now()
	}
	notAfter := m.caCert.NotAfter
	totalDuration := float64(notAfter.Sub(m.caCert.NotBefore))
	deadline := m.caCert.NotBefore.Add(jitteryDuration(totalDuration))

	m.log.Info(fmt.Sprintf("Certificate expiration is %v, rotation deadline is %v", notAfter, deadline))
	return deadline
}

// jitteryDuration uses some jitter to set the rotation threshold so each node
// will rotate at approximately 70-90% of the total lifetime of the
// certificate.  With jitter, if a number of nodes are added to a cluster at
// approximately the same time (such as cluster creation time), they won't all
// try to rotate certificates at the same time for the rest of the life of the
// cluster.
//
// This function is represented as a variable to allow replacement during testing.
var jitteryDuration = func(totalDuration float64) time.Duration {
	return wait.Jitter(time.Duration(totalDuration), 0.2) - time.Duration(totalDuration*0.3)
}
