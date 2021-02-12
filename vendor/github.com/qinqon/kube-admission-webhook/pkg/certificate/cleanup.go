package certificate

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

func (m *Manager) earliestElapsedForCACertsCleanup() (time.Duration, error) {
	cas, err := m.getCACertsFromCABundle()
	if err != nil {
		return time.Duration(0), errors.Wrap(err, "failed getting CA certificates from CA bundle")
	}
	return m.earliestElapsedForCleanup(m.log.WithName("earliestElapsedForCACertsCleanup"), cas)
}

// earliestElapsedForServiceCertsCleanup will iterate all the services and
// retrieve the secrets associate, calculate the elapsed time for
// cleanup for each and return the min.
func (m *Manager) earliestElapsedForServiceCertsCleanup() (time.Duration, error) {
	webhookConf, err := m.readyWebhookConfiguration()
	if err != nil {
		return time.Duration(0), fmt.Errorf("failed getting webhook configuration to calculate cleanup next run: %w", err)
	}

	services, err := m.getServicesFromConfiguration(webhookConf)
	if err != nil {
		return time.Duration(0), fmt.Errorf("failed getting services to calculate cleanup next run: %w", err)
	}

	elapsedTimesForCleanup := []time.Duration{}
	for service, _ := range services {

		certs, err := m.getTLSCerts(service)
		if err != nil {
			return time.Duration(0), fmt.Errorf("failed getting TLS keypair from service %s to calculate cleanup next run: %w", service, err)
		}
		elapsedTimeForCleanup, err := m.earliestElapsedForCleanup(m.log.WithName("earliestElapsedForServiceCertsCleanup").WithValues("service", service), certs)
		if err != nil {
			return time.Duration(0), err
		}
		elapsedTimesForCleanup = append(elapsedTimesForCleanup, elapsedTimeForCleanup)
	}
	return min(elapsedTimesForCleanup...), nil
}

// earliestElapsedForCleanup return a subtraction between earliestCleanupDeadline and
// `now`
func (m *Manager) earliestElapsedForCleanup(log logr.Logger, certificates []*x509.Certificate) (time.Duration, error) {
	deadline := m.earliestCleanupDeadlineForCerts(certificates)
	now := m.now()
	elapsedForCleanup := deadline.Sub(now)
	log.Info(fmt.Sprintf("{now: %s, deadline: %s, elapsedForCleanup: %s}", now, deadline, elapsedForCleanup))
	return elapsedForCleanup, nil
}

// earliestCleanupDeadlineForCACerts will inspect CA certificates
// select the deadline based on expiration time
func (m *Manager) earliestCleanupDeadlineForCerts(certificates []*x509.Certificate) time.Time {
	var selectedCertificate *x509.Certificate

	// There is no overlap just return expiration time
	if len(certificates) == 1 {
		return certificates[0].NotAfter
	}

	for _, certificate := range certificates {
		if selectedCertificate == nil || certificate.NotAfter.Before(selectedCertificate.NotAfter) {
			selectedCertificate = certificate
		}
	}
	if selectedCertificate == nil {
		return m.now()
	}
	return selectedCertificate.NotAfter
}

func (m *Manager) cleanUpCABundle() error {
	m.log.Info("cleanUpCABundle")
	_, err := m.updateWebhookCABundleWithFunc(func([]byte) ([]byte, error) {
		cas, err := m.getCACertsFromCABundle()
		if err != nil {
			return nil, errors.Wrap(err, "failed getting ca certs to start cleanup")
		}
		cleanedCAs := m.cleanUpCertificates(cas)
		pem := triple.EncodeCertsPEM(cleanedCAs)
		return pem, nil
	})

	if err != nil {
		return errors.Wrap(err, "failed updating webhook config after ca certificates cleanup")
	}
	return nil
}

func (m *Manager) cleanUpServiceCerts() error {
	m.log.Info("cleanUpServiceCerts")
	webhookConf, err := m.readyWebhookConfiguration()
	if err != nil {
		return fmt.Errorf("failed getting webhook configuration to do the cleanup: %w", err)
	}

	services, err := m.getServicesFromConfiguration(webhookConf)
	if err != nil {
		return fmt.Errorf("failed getting services to do the cleanup: %w", err)
	}

	for service, _ := range services {
		m.applySecret(service, corev1.SecretTypeTLS, nil, func(secret corev1.Secret, keyPair *triple.KeyPair) (*corev1.Secret, error) {
			certPEM, found := secret.Data[corev1.TLSCertKey]
			if !found {
				return nil, errors.Wrapf(err, "TLS cert not found at secret %s to clean up ", service)
			}

			certs, err := triple.ParseCertsPEM(certPEM)
			if err != nil {
				return nil, errors.Wrapf(err, "failed parsing TLS cert PEM at secret %s to clean up", service)
			}

			cleanedCerts := m.cleanUpCertificates(certs)
			pem := triple.EncodeCertsPEM(cleanedCerts)
			secret.Data[corev1.TLSCertKey] = pem
			return &secret, nil
		})
	}
	return nil
}

func (m *Manager) cleanUpCertificates(certificates []*x509.Certificate) []*x509.Certificate {
	logger := m.log.WithName("cleanUpCertificates")
	// There is no overlap
	if len(certificates) <= 1 {
		return certificates
	}

	now := m.now()
	// create a zero-length slice with the same underlying array
	cleanedUpCertificates := certificates[:0]
	for _, certificate := range certificates {
		logger.Info("Checking certificate for cleanup", "now", now, "NotBefore", certificate.NotBefore, "NotAfter", certificate.NotAfter)

		// Expired certificate are cleaned up
		expirationDate := certificate.NotAfter
		if now.Equal(expirationDate) || now.After(expirationDate) {
			logger.Info("Cleaning up expired certificate", "now", now, "NotBefore", certificate.NotBefore, "NotAfter", certificate.NotAfter)
			continue
		}

		cleanedUpCertificates = append(cleanedUpCertificates, certificate)
	}
	return cleanedUpCertificates
}
