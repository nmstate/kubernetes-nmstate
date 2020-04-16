package certificate

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	certificatesclientv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	"k8s.io/client-go/util/certificate"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type manager struct {
	crMgr            crmanager.Manager
	certManager      certificate.Manager
	certStore        *filePairStore
	log              logr.Logger
	caConfigMapKey   types.NamespacedName
	caConfigMapField string
}

type WebhookType string

const (
	MutatingWebhook   WebhookType = "Mutating"
	ValidatingWebhook WebhookType = "Validating"
)

// NewManager with create a certManager that generated a pair of files ${certDir}/${certFile} and ${certDir}/${keyFile} to use them
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
	crMgr crmanager.Manager,
	webhookName string,
	webhookType WebhookType,
	certDir string, certFileName string, keyFileName string, caConfigMapKey types.NamespacedName, caConfigMapField string) (*manager, error) {

	certStore, err := NewFilePairStore(
		certDir,
		certFileName,
		keyFileName,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize new webhook cert/key file store")
	}

	m := &manager{
		crMgr:            crMgr,
		log:              logf.Log.WithName("webhook/server/certificate/manager"),
		certStore:        certStore,
		caConfigMapKey:   caConfigMapKey,
		caConfigMapField: caConfigMapField,
	}

	dnsNames := []string{}
	if webhookType == MutatingWebhook {
		mutatingWebHookConfig, err := m.updateMutatingWebhookCABundle(webhookName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update CA bundle at webhook")
		}

		for _, webhook := range mutatingWebHookConfig.Webhooks {
			dnsNames = append(dnsNames, dnsNamesForService(*webhook.ClientConfig.Service)...)
		}
	} else if webhookType == ValidatingWebhook {
		validatingWebHookConfig, err := m.updateValidatingWebhookCABundle(webhookName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update CA bundle at webhook")
		}

		for _, webhook := range validatingWebHookConfig.Webhooks {
			dnsNames = append(dnsNames, dnsNamesForService(*webhook.ClientConfig.Service)...)
		}
	}

	certConfig := certificate.Config{
		ClientFn: func(current *tls.Certificate) (certificatesclientv1beta1.CertificateSigningRequestInterface, error) {
			certClient, err := certificatesclientv1beta1.NewForConfig(crMgr.GetConfig())
			if err != nil {
				return nil, errors.Wrap(err, "failed to create cert client for webhook")
			}
			return newCSRApprover(certClient.CertificateSigningRequests()), nil
		},
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: webhookName,
			},
			DNSNames: dnsNames,
		},
		Usages: []certificatesv1beta1.KeyUsage{
			certificatesv1beta1.UsageDigitalSignature,
			certificatesv1beta1.UsageKeyEncipherment,
			certificatesv1beta1.UsageServerAuth,
		},
		CertificateStore: certStore,
	}

	certManager, err := certificate.NewManager(&certConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed initializing webhook cert manager")
	}

	m.certManager = certManager
	return m, nil
}

func dnsNamesForService(service admissionregistrationv1beta1.ServiceReference) []string {
	return []string{
		fmt.Sprintf("%s", service.Name),
		fmt.Sprintf("%s.%s", service.Name, service.Namespace),
		fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace),
	}
}

// Will start the the underlaying client-go cert manager [1]  and
// wait for TLS key and cert to be generated
//
// [1] https://godoc.org/k8s.io/client-go/util/certificate
func (m manager) Start() error {
	m.log.Info("Starting cert manager")
	m.certManager.Start()

	m.log.Info("Wait for cert/key to be created")
	err := wait.PollImmediate(time.Second, 120*time.Second, func() (bool, error) {
		keyExists, err := m.certStore.keyFileExists()
		if err != nil {
			return false, err
		}
		certExists, err := m.certStore.keyFileExists()
		if err != nil {
			return false, err
		}
		return keyExists && certExists, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed creating webhook tls key/cert")
	}
	m.log.Info(fmt.Sprintf("TLS cert/key ready at %s", m.certStore.CurrentPath()))

	certificate, err := m.certStore.Current()
	if err != nil {
		return errors.Wrap(err, "failed retrieving webhook current certificate")
	}
	m.log.Info(fmt.Sprintf("Certificate expiration is %v-%v", certificate.Leaf.NotBefore, certificate.Leaf.NotAfter))

	return nil
}

func (m manager) Stop() {
	m.log.Info("Stopping cert manager")
	m.certManager.Stop()
}
