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
// select the deadline based on certificate NotBefore + caOverlapDuration
// returning the daedline that is going to happend sooner
func (m *Manager) earliestCleanupDeadlineForCerts(certificates []*x509.Certificate) time.Time {
	var selectedCertificate *x509.Certificate

	// There is no overlap just return expiration time
	if len(certificates) == 1 {
		return certificates[0].NotAfter
	}

	for _, certificate := range certificates {
		if selectedCertificate == nil || certificate.NotBefore.Before(selectedCertificate.NotBefore) {
			selectedCertificate = certificate
		}
	}
	if selectedCertificate == nil {
		return m.now()
	}

	// Add the overlap duration since is the time certs are going to be living
	// add CABundle
	return selectedCertificate.NotBefore.Add(m.caOverlapDuration)
}

func (m *Manager) cleanUpCABundle() error {
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

func (m *Manager) cleanUpCertificates(certificates []*x509.Certificate) []*x509.Certificate {
	logger := m.log.WithName("cleanUpCertificates")
	logger.Info("Cleaning up expired or beyond overlap duration limit at CA bundle")
	// There is no overlap
	if len(certificates) <= 1 {
		return certificates
	}

	now := m.now()
	// create a zero-length slice with the same underlying array
	cleanedUpCertificates := certificates[:0]
	for i, certificate := range certificates {
		logger.Info("Checking certificate for cleanup", "now", now, "caOverlapDuration", m.caOverlapDuration, "NotBefore", certificate.NotBefore, "NotAfter", certificate.NotAfter)

		// Expired certificate are cleaned up
		caExpirationDate := certificate.NotAfter
		if now.After(caExpirationDate) {
			logger.Info("Cleaning up expired certificate", "now", now, "NotBefore", certificate.NotBefore, "NotAfter", certificate.NotAfter)
			continue
		}

		// Clean up certificates that pass CA Overlap Duration limit,
		// except for the last appended one (i.e. the last generated from a rotation) since we need at least one valid certificate
		caOverlapDate := certificate.NotBefore.Add(m.caOverlapDuration)
		if i != len(certificates)-1 && !now.Before(caOverlapDate) {
			logger.Info("Cleaning up certificate beyond CA overlap duration", "now", now, "caOverlapDuration", m.caOverlapDuration, "NotBefore", certificate.NotBefore, "NotAfter", certificate.NotAfter)
			continue
		}
		cleanedUpCertificates = append(cleanedUpCertificates, certificate)
	}
	return cleanedUpCertificates
}
