package certificate

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

// earliestElapsedForCleanup return a subtraction between earliestCleanupDeadline and
// `now`
func (m *Manager) earliestElapsedForCleanup() (time.Duration, error) {
	deadline, err := m.earliestCleanupDeadline()
	if err != nil {
		return time.Duration(0), errors.Wrap(err, "failed calculating cleanup deadline")
	}
	now := m.now()
	elapsedForCleanup := deadline.Sub(now)
	m.log.Info(fmt.Sprintf("earliestElapsedForCleanup {now: %s, deadline: %s, elapsedForCleanup: %s}", now, deadline, elapsedForCleanup))
	return elapsedForCleanup, nil
}

// earliestCleanupDeadline get all the certificates at CABundle and return the
// deadline calculated by earliestCleanupDeadlineForCerts
func (m *Manager) earliestCleanupDeadline() (time.Time, error) {
	cas, err := m.getCACertsFromCABundle()
	if err != nil {
		return m.now(), errors.Wrap(err, "failed getting CA certificates from CA bundle")
	}

	return m.earliestCleanupDeadlineForCerts(cas), nil
}

// earliestCleanupDeadlineForCACerts will inspect CA certificates
// select the deadline based on certificate that is going to expire
// sooner so cleanup is triggered then
func (m *Manager) earliestCleanupDeadlineForCerts(certificates []*x509.Certificate) time.Time {

	var selectedCertificate *x509.Certificate

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
	logger := m.log.WithName("cleanUpCABundle")
	logger.Info("Cleaning up expired certificates at CA bundle")
	_, err := m.updateWebhookCABundleWithFunc(func([]byte) ([]byte, error) {
		cas, err := m.getCACertsFromCABundle()
		if err != nil {
			return nil, errors.Wrap(err, "failed getting ca certs to start cleanup")
		}
		cleanedCAs := m.cleanUpExpiredCertificates(cas)
		pem := triple.EncodeCertsPEM(cleanedCAs)
		return pem, nil
	})

	if err != nil {
		return errors.Wrap(err, "failed updating webhook config after ca certificates cleanup")
	}
	return nil
}

func (m *Manager) cleanUpExpiredCertificates(certificates []*x509.Certificate) []*x509.Certificate {
	now := m.now()
	// create a zero-length slice with the same underlying array
	cleanedUpCertificates := certificates[:0]
	for _, certificate := range certificates {
		if certificate.NotAfter.After(now) {
			cleanedUpCertificates = append(cleanedUpCertificates, certificate)
		}
	}
	return cleanedUpCertificates
}
