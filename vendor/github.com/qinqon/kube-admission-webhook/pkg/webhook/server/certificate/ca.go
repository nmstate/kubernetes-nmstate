package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func (m manager) get(key types.NamespacedName, value runtime.Object) error {
	return wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
		err := m.crMgr.GetClient().Get(context.TODO(), key, value)
		if err != nil {
			_, cacheNotStarted := err.(*cache.ErrCacheNotStarted)
			if cacheNotStarted {
				return false, nil
			} else {
				return true, err
			}
		}
		return true, nil
	})
}

// Retrieve cluster CA bundle and encode to base 64
func (m manager) clientCAFile() ([]byte, error) {
	authenticationConfig := corev1.ConfigMap{}
	err := m.get(types.NamespacedName{Namespace: "kube-system", Name: "extension-apiserver-authentication"}, &authenticationConfig)
	if err != nil {
		return []byte{}, errors.Wrap(err, "failed to retrieve cluster authentication config")
	}

	clientCaFile := authenticationConfig.Data["client-ca-file"]
	return []byte(clientCaFile), nil
}

func mutatingWebhookConfig(webhook runtime.Object) *admissionregistrationv1beta1.MutatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1beta1.MutatingWebhookConfiguration)
}

func validatingWebhookConfig(webhook runtime.Object) *admissionregistrationv1beta1.ValidatingWebhookConfiguration {
	return webhook.(*admissionregistrationv1beta1.ValidatingWebhookConfiguration)
}

// clientConfigList returns the the list of webhooks's mutation or validationg clientConfig, clientConfig is the information at the webhook config pointing to the service and path [1].
//
//  [1] https://godoc.org/k8s.io/kubernetes/pkg/apis/admissionregistration#WebhookClientConfig
func clientConfigList(webhook runtime.Object, webhookType WebhookType) []*admissionregistrationv1beta1.WebhookClientConfig {
	clientConfigList := []*admissionregistrationv1beta1.WebhookClientConfig{}
	if webhookType == MutatingWebhook {
		mutatingWebhookConfig := mutatingWebhookConfig(webhook)
		for i, _ := range mutatingWebhookConfig.Webhooks {
			clientConfig := &mutatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	} else if webhookType == ValidatingWebhook {
		validatingWebhookConfig := validatingWebhookConfig(webhook)
		for i, _ := range validatingWebhookConfig.Webhooks {
			clientConfig := &validatingWebhookConfig.Webhooks[i].ClientConfig
			clientConfigList = append(clientConfigList, clientConfig)
		}
	}
	return clientConfigList
}

func (m manager) updateWebhookCABundle(webhookName string, webhookType WebhookType) (runtime.Object, error) {
	m.log.Info("Updating CA bundle for webhook")

	var webhook runtime.Object
	if webhookType == MutatingWebhook {
		webhook = &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
	} else if webhookType == ValidatingWebhook {
		webhook = &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
	} else {
		return nil, fmt.Errorf("Unknown webhook type %s", webhookType)
	}

	clientCAFile, err := m.clientCAFile()
	if err != nil {
		return webhook, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Do some polling to wait for manifest to be deployed
		err := wait.PollImmediate(1*time.Second, 120*time.Second, func() (bool, error) {
			webhookKey := types.NamespacedName{Name: webhookName}
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
			return errors.Wrapf(err, "failed retrieving %s webhook %s", webhookType, webhookName)
		}

		for _, clientConfig := range clientConfigList(webhook, webhookType) {
			// Update the CA bundle at webhook
			clientConfig.CABundle = []byte(clientCAFile)
		}

		err = m.crMgr.GetClient().Update(context.TODO(), webhook)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return webhook, errors.Wrap(err, "failed to update validating webhook CABundle")
	}
	return webhook, nil
}

func (m manager) updateValidatingWebhookCABundle(webhookName string) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
	webhook, err := m.updateWebhookCABundle(webhookName, ValidatingWebhook)
	if err != nil {
		return nil, err
	}
	return validatingWebhookConfig(webhook), nil
}

func (m manager) updateMutatingWebhookCABundle(webhookName string) (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
	webhook, err := m.updateWebhookCABundle(webhookName, MutatingWebhook)
	if err != nil {
		return nil, err
	}
	return mutatingWebhookConfig(webhook), nil
}
