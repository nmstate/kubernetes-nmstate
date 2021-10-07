package certificate

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

func mutatingWebhookConfig(webhook runtime.Object) *admissionregistrationv1beta1.MutatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1beta1.MutatingWebhookConfiguration)
}

func validatingWebhookConfig(webhook runtime.Object) *admissionregistrationv1beta1.ValidatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration)
}

// clientConfigList returns the the list of webhooks's mutation or validating WebhookClientConfig
//
// The WebhookClientConfig type is share between mutating or validating so we can have a common function
// that uses the interface runtime.Object and do some type checking to access it [1].
//
// [1] https://godoc.org/k8s.io/kubernetes/pkg/apis/admissionregistration#WebhookClientConfig
func (m *Manager) clientConfigList(webhook runtime.Object) []*admissionregistrationv1beta1.WebhookClientConfig {
	clientConfigList := []*admissionregistrationv1beta1.WebhookClientConfig{}
	if m.webhookType == MutatingWebhook {
		mutatingWebhookConfig := mutatingWebhookConfig(webhook)
		for i, _ := range mutatingWebhookConfig.Webhooks {
			clientConfig := &mutatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	} else if m.webhookType == ValidatingWebhook {
		validatingWebhookConfig := validatingWebhookConfig(webhook)
		for i, _ := range validatingWebhookConfig.Webhooks {
			clientConfig := &validatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	}
	return clientConfigList
}

func (m *Manager) readyWebhookConfiguration() (runtime.Object, error) {
	var webhook runtime.Object
	if m.webhookType == MutatingWebhook {
		webhook = &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
	} else if m.webhookType == ValidatingWebhook {
		webhook = &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
	} else {
		return nil, fmt.Errorf("Unknown webhook type %s", m.webhookType)
	}

	// Do some polling to wait for manifest to be deployed
	err := wait.PollImmediate(1*time.Second, 120*time.Second, func() (bool, error) {
		webhookKey := types.NamespacedName{Name: m.webhookName}
		err := m.get(webhookKey, webhook)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed retrieving %s webhook %s", m.webhookType, m.webhookName)
	}
	return webhook, err
}

func (m *Manager) addCertificateToCABundle(caCert *x509.Certificate) error {
	m.log.Info("Reset CA bundle with one cert for webhook")
	_, err := m.updateWebhookCABundleWithFunc(func(currentCABundle []byte) ([]byte, error) {
		cas := []*x509.Certificate{}
		if len(currentCABundle) > 0 {
			var err error
			cas, err = triple.ParseCertsPEM(currentCABundle)
			if err != nil {
				return nil, errors.Wrap(err, "failed parsing current CA bundle")
			}
		}
		cas = append(cas, caCert)
		cas = triple.RemoveOldestCerts(cas, triple.CertsListSizeLimit)
		return triple.EncodeCertsPEM(cas), nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to update webhook CABundle")
	}
	return nil
}

func (m *Manager) updateWebhookCABundleWithFunc(updateCABundle func([]byte) ([]byte, error)) (runtime.Object, error) {
	m.log.Info("Updating CA bundle for webhook")
	var webhook runtime.Object
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		webhook, err = m.readyWebhookConfiguration()
		if err != nil {
			return errors.Wrapf(err, "failed to get %s webhook configuration %s", m.webhookType, m.webhookName)
		}

		for _, clientConfig := range m.clientConfigList(webhook) {
			// Update the CA bundle at webhook
			updatedCABundle, err := updateCABundle(clientConfig.CABundle)
			if err != nil {
				return errors.Wrap(err, "failed updating CA bundle")
			}
			clientConfig.CABundle = updatedCABundle
		}

		err = m.client.Update(context.TODO(), webhook)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update webhook CABundle")
	}
	return webhook, nil
}

func (m *Manager) CABundle() ([]byte, error) {
	webhook, err := m.readyWebhookConfiguration()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s webhook configuration %s", m.webhookType, m.webhookName)
	}

	clientConfigList := m.clientConfigList(webhook)
	if clientConfigList == nil || len(clientConfigList) == 0 {
		return nil, errors.Wrapf(err, "failed to access CABundle clientConfigList is empty in %s webhook configuration %s", m.webhookType, m.webhookName)
	}

	return clientConfigList[0].CABundle, nil
}

// getServicesFromConfiguration it retrieves all the references services at
// webhook configuration clientConfig and in case there is URL instead of
// ServiceRef it will reference fake one with webhook name, mgr namespace and
// passing the url hostname at map value
func (m *Manager) getServicesFromConfiguration(configuration runtime.Object) (map[types.NamespacedName][]string, error) {

	logger := m.log.WithName("getServicesFromConfiguration")
	services := map[types.NamespacedName][]string{}

	for _, clientConfig := range m.clientConfigList(configuration) {

		service := types.NamespacedName{}
		hostnames := []string{}

		if clientConfig.Service != nil {
			logger.Info("Composing service name and namespace from ServiceRef", "serviceRef", clientConfig.Service)
			service.Name = clientConfig.Service.Name
			service.Namespace = clientConfig.Service.Namespace
		} else if clientConfig.URL != nil {
			logger.Info("Composing service name and namespace from URL", "URL", clientConfig.URL)
			service.Name = m.webhookName
			service.Namespace = m.namespace
			u, err := url.Parse(*clientConfig.URL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed parsing webhook URL %s", *clientConfig.URL)
			}
			hostnames = append(hostnames, strings.Split(u.Host, ":")[0])
		} else {
			return nil, errors.New("bad configuration, webhook without serviceRef or URL")
		}

		services[service] = hostnames
	}
	return services, nil
}
