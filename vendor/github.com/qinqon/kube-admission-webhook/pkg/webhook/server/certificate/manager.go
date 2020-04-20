package certificate

import (
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

type Manager struct {
	client        client.Client
	webhookName   string
	webhookType   WebhookType
	caKeyPair     *triple.KeyPair
	now           func() time.Time
	certsDuration time.Duration
	log           logr.Logger
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

// Start the cert manager until stopCh is close, the cert manager is in charge
// of rotate certificate if needed.
//
// It  implemenets Runnable [1] so manager can add this to a
// controller runtime manager
//
// [1] https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/manager/manager.go#L208
func (m *Manager) Start(stopCh <-chan struct{}) error {
	m.log.Info("Starting cert manager")

	wait.Until(func() {
		m.waitForDeadlineAndRotate()
	}, time.Second, stopCh)

	return nil
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
	err := wait.PollImmediateInfinite(32*time.Second, m.rotateCondition)
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

func (m *Manager) rotateCondition() (bool, error) {
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

	m.caKeyPair = caKeyPair

	err = m.updateWebhookCABundle()
	if err != nil {
		return errors.Wrap(err, "failed to update CA bundle at webhook")
	}

	webhookConf, err := m.webhookConfiguration()
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
		m.createOrUpdateTLSSecret(service, keyPair)
	}

	return nil
}

// nextRotationDeadline returns a value for the threshold at which the
// current certificate should be rotated, 80%+/-10% of the expiration of the
// certificate.
func (m *Manager) nextRotationDeadline() time.Time {
	if m.caKeyPair == nil {
		m.log.Info("Certificates not created, forcing roration")
		return m.now()
	}
	notAfter := m.caKeyPair.Cert.NotAfter
	totalDuration := float64(notAfter.Sub(m.caKeyPair.Cert.NotBefore))
	deadline := m.caKeyPair.Cert.NotBefore.Add(jitteryDuration(totalDuration))

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
