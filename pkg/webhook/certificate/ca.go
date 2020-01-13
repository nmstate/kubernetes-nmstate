package certificate

import (
	"context"
	"time"

	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// Retrieve cluster CA bundle and encode to base 64
func (m manager) clientCAFile() ([]byte, error) {
	authenticationConfig := corev1.ConfigMap{}
	err := m.crMgr.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: "kube-system", Name: "extension-apiserver-authentication"}, &authenticationConfig)

	if err != nil {
		return []byte{}, errors.Wrap(err, "failed to retrieve cluster authentication config")
	}
	clientCaFile := authenticationConfig.Data["client-ca-file"]
	return []byte(clientCaFile), nil
}

func (m manager) updateMutatingWebhookCABundle(webhookName string) (admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
	m.log.Info("Updating CA bundle for webhook")
	mutatingWebHook := admissionregistrationv1beta1.MutatingWebhookConfiguration{}

	clientCAFile, err := m.clientCAFile()
	if err != nil {
		return mutatingWebHook, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Do some polling to wait for manifest to be deployed
		err := wait.PollImmediate(1*time.Second, 120*time.Second, func() (bool, error) {
			webhookKey := types.NamespacedName{Name: webhookName}
			err := m.crMgr.GetClient().Get(context.TODO(), webhookKey, &mutatingWebHook)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		})

		if err != nil {
			return errors.Wrap(err, "failed retrieving mutationg webhook "+webhookName)
		}

		for i, _ := range mutatingWebHook.Webhooks {
			// Update the CA bundle at webhook
			mutatingWebHook.Webhooks[i].ClientConfig.CABundle = []byte(clientCAFile)
		}

		err = m.crMgr.GetClient().Update(context.TODO(), &mutatingWebHook)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mutatingWebHook, errors.Wrap(err, "failed to update mutating webhook CABundle")
	}
	return mutatingWebHook, nil
}
