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
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

func mutatingWebhookConfig(webhook client.Object) *admissionregistrationv1.MutatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1.MutatingWebhookConfiguration)
}

func validatingWebhookConfig(webhook client.Object) *admissionregistrationv1.ValidatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1.ValidatingWebhookConfiguration)
}

// clientConfigList returns the the list of webhooks's mutation or validating WebhookClientConfig
//
// The WebhookClientConfig type is share between mutating or validating so we can have a common function
// that uses the interface client.Object and do some type checking to access it [1].
//
// [1] https://godoc.org/k8s.io/kubernetes/pkg/apis/admissionregistration#WebhookClientConfig
func (m *Manager) clientConfigList(webhook client.Object) []*admissionregistrationv1.WebhookClientConfig {
	clientConfigList := []*admissionregistrationv1.WebhookClientConfig{}
	if m.webhookType == MutatingWebhook {
		mutatingWebhookConfig := mutatingWebhookConfig(webhook)
		for i := range mutatingWebhookConfig.Webhooks {
			clientConfig := &mutatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	} else if m.webhookType == ValidatingWebhook {
		validatingWebhookConfig := validatingWebhookConfig(webhook)
		for i := range validatingWebhookConfig.Webhooks {
			clientConfig := &validatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	}
	return clientConfigList
}

func (m *Manager) readyWebhookConfiguration() (client.Object, error) {
	var webhook client.Object
	if m.webhookType == MutatingWebhook {
		webhook = &admissionregistrationv1.MutatingWebhookConfiguration{}
	} else if m.webhookType == ValidatingWebhook {
		webhook = &admissionregistrationv1.ValidatingWebhookConfiguration{}
	} else {
		return nil, fmt.Errorf("unknown webhook type %s", m.webhookType)
	}
	const (
		pollInterval = time.Second
		pollTimeout  = 120 * time.Second
	)
	// Do some polling to wait for manifest to be deployed
	err := wait.PollUntilContextTimeout(context.TODO(), pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
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
	err := m.updateWebhookCABundleWithFunc(func(currentCABundle []byte) ([]byte, error) {
		return triple.AddCertToPEM(caCert, currentCABundle, triple.CertsListSizeLimit)
	})
	if err != nil {
		return errors.Wrap(err, "failed to update webhook CABundle")
	}
	return nil
}

func (m *Manager) updateWebhookCABundleWithFunc(updateCABundle func([]byte) ([]byte, error)) error {
	m.log.Info("Updating CA bundle for webhook")
	var webhook client.Object
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		webhook, err = m.readyWebhookConfiguration()
		if err != nil {
			return errors.Wrapf(err, "failed to get %s webhook configuration %s", m.webhookType, m.webhookName)
		}

		for _, clientConfig := range m.clientConfigList(webhook) {
			// Update the CA bundle at webhook
			var updatedCABundle []byte
			updatedCABundle, err = updateCABundle(clientConfig.CABundle)
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
		return errors.Wrap(err, "failed to update webhook CABundle")
	}
	return nil
}

func (m *Manager) CABundle() ([]byte, error) {
	webhook, err := m.readyWebhookConfiguration()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s webhook configuration %s", m.webhookType, m.webhookName)
	}

	clientConfigList := m.clientConfigList(webhook)
	if len(clientConfigList) == 0 {
		return nil, errors.Wrapf(err,
			"failed to access CABundle clientConfigList is empty in %s webhook configuration %s", m.webhookType, m.webhookName)
	}

	return clientConfigList[0].CABundle, nil
}

// getServicesFromConfiguration it retrieves all the references services at
// webhook configuration clientConfig and in case there is URL instead of
// ServiceRef it will reference fake one with webhook name, mgr namespace and
// passing the url hostname at map value
func (m *Manager) getServicesFromConfiguration(configuration client.Object) (map[types.NamespacedName][]string, error) {
	services := map[types.NamespacedName][]string{}

	for _, clientConfig := range m.clientConfigList(configuration) {
		service := types.NamespacedName{}
		hostnames := []string{}

		if clientConfig.Service != nil {
			service.Name = clientConfig.Service.Name
			service.Namespace = clientConfig.Service.Namespace
		} else if clientConfig.URL != nil {
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
