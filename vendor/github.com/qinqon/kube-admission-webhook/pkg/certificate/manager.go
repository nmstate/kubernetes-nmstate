package certificate

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
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

	// now is an artifact to do some unit testing without waiting for
	// expiration time.
	now func() time.Time

	// lastRotateDeadline store the value of last call from nextRotationDeadline
	lastRotateDeadline *time.Time

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
		log: logf.Log.WithName("certificate/manager").
			WithValues("webhookType", webhookType, "webhookName", webhookName),
	}
	return m
}

func (m *Manager) getCACertsFromCABundle() ([]*x509.Certificate, error) {
	caBundle, err := m.CABundle()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting CABundle")
	}

	if len(caBundle) == 0 {
		return nil, nil
	}

	cas, err := triple.ParseCertsPEM(caBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing PEM CABundle")
	}
	return cas, nil
}

func (m *Manager) getLastAppendedCACertFromCABundle() (*x509.Certificate, error) {
	cas, err := m.getCACertsFromCABundle()
	if err != nil {
		return nil, errors.Wrap(err, "failed getting CA certificates from CA bundle")
	}
	if len(cas) == 0 {
		return nil, nil
	}
	return cas[len(cas)-1], nil
}

func (m *Manager) rotate() error {

	m.log.Info("Rotating TLS cert/key")

	caKeyPair, err := triple.NewCA(m.webhookName, m.certsDuration)
	if err != nil {
		return errors.Wrap(err, "failed generating CA cert/key")
	}

	webhook, err := m.addCertificateToCABundle(caKeyPair.Cert)
	if err != nil {
		return errors.Wrap(err, "failed adding new CA cert to CA bundle at webhook")
	}

	for _, clientConfig := range m.clientConfigList(webhook) {

		service := types.NamespacedName{}
		hostnames := []string{}

		if clientConfig.Service != nil {
			service.Name = clientConfig.Service.Name
			service.Namespace = clientConfig.Service.Namespace
		} else if clientConfig.URL != nil {
			service.Name = m.webhookName
			service.Namespace = "default"
			u, err := url.Parse(*clientConfig.URL)
			if err != nil {
				return errors.Wrapf(err, "failed parsing webhook URL %s", *clientConfig.URL)
			}
			hostnames = append(hostnames, strings.Split(u.Host, ":")[0])
		}

		keyPair, err := triple.NewServerKeyPair(
			caKeyPair,
			service.Name+"."+service.Namespace+".pod.cluster.local",
			service.Name,
			service.Namespace,
			"cluster.local",
			nil,
			hostnames,
			m.certsDuration,
		)
		if err != nil {
			return errors.Wrapf(err, "failed creating server key/cert for service %+v", service)
		}
		err = m.applyTLSSecret(service, keyPair)
		if err != nil {
			return errors.Wrapf(err, "failed applying TLS secret %s", service)
		}
	}

	return nil
}

// nextRotationDeadline returns a value for the threshold at which the
// current certificate should be rotated, 80%+/-10% of the expiration of the
// certificate or force rotation in case the certificate chain is faulty
func (m *Manager) nextRotationDeadline() time.Time {
	err := m.verifyTLS()
	if err != nil {
		// Sprintf is used to prevent stack trace to be printed
		m.log.Info(fmt.Sprintf("Bad TLS certificate chain, forcing rotation: %v", err))
		return m.now()
	}

	// Last rotated CA cert at CABundle is the last at the slice so this
	// calculate deadline from it.
	caCert, err := m.getLastAppendedCACertFromCABundle()
	if err != nil {
		m.log.Info("Failed reading last CA cert from CABundle, forcing rotation", "err", err)
		return m.now()
	}
	nextDeadline := m.nextRotationDeadlineForCert(caCert)

	// Store last calculated deadline to use it at Reconcile
	m.lastRotateDeadline = &nextDeadline
	return nextDeadline
}

// nextRotationDeadlineForCert returns a value for the threshold at which the
// current certificate should be rotated, 80%+/-10% of the expiration of the
// certificate
func (m *Manager) nextRotationDeadlineForCert(certificate *x509.Certificate) time.Time {
	notAfter := certificate.NotAfter
	totalDuration := float64(notAfter.Sub(certificate.NotBefore))
	deadline := certificate.NotBefore.Add(jitteryDuration(totalDuration))

	m.log.Info(fmt.Sprintf("Certificate expiration is %v, totalDuration is %v, rotation deadline is %v", notAfter, totalDuration, deadline))
	return deadline
}

func (m *Manager) elapsedToRotateFromLastDeadline() time.Duration {
	deadline := m.now()

	// If deadline was previously calculated return it, else do the
	// calculations
	if m.lastRotateDeadline != nil {
		deadline = *m.lastRotateDeadline
	} else {
		deadline = m.nextRotationDeadline()
	}
	now := m.now()
	elapsedToRotate := deadline.Sub(now)
	m.log.Info(fmt.Sprintf("elapsedToRotateFromLastDeadline {now: %s, deadline: %s, elapsedToRotate: %s}", now, deadline, elapsedToRotate))
	return elapsedToRotate
}

// verifyTLS will verify that the caBundle and Secret are valid and can
// be used to verify
func (m *Manager) verifyTLS() error {

	webhookConf, err := m.readyWebhookConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to reading configuration")
	}

	for _, clientConfig := range m.clientConfigList(webhookConf) {
		service := clientConfig.Service
		secretKey := types.NamespacedName{}
		if service != nil {
			// If the webhook has a service then create the secret
			// with same namespce and name
			secretKey.Name = service.Name
			secretKey.Namespace = service.Namespace
		} else {
			// If it uses directly URL create a secret with webhookName and
			// default namespace
			secretKey.Name = m.webhookName
			secretKey.Namespace = "default"
		}
		err = m.verifyTLSSecret(secretKey, clientConfig.CABundle)
		if err != nil {
			return errors.Wrapf(err, "failed verifying TLS secret %s", secretKey)
		}
	}

	return nil
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
