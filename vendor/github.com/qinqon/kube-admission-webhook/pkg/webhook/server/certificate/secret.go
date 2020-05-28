package certificate

import (
	"context"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate/triple"
)

func updateTLSSecret(secret corev1.Secret, keyPair *triple.KeyPair) *corev1.Secret {
	secret.Data = map[string][]byte{
		corev1.TLSCertKey:       triple.EncodeCertPEM(keyPair.Cert),
		corev1.TLSPrivateKeyKey: triple.EncodePrivateKeyPEM(keyPair.Key),
	}
	secret.Type = corev1.SecretTypeTLS
	return &secret
}

func (m *Manager) newTLSSecret(serviceKey types.NamespacedName, keyPair *triple.KeyPair) (*corev1.Secret, error) {
	service := corev1.Service{}
	err := m.get(serviceKey, &service)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting service %s to set secret owner", serviceKey)
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       service.Name,
					Kind:       service.TypeMeta.Kind,
					APIVersion: service.TypeMeta.APIVersion,
					UID:        service.UID},
			},
		},
	}
	return updateTLSSecret(secret, keyPair), nil
}

func (m *Manager) applyTLSSecret(service types.NamespacedName, keyPair *triple.KeyPair) error {
	secret := corev1.Secret{}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := m.get(service, &secret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				tlsSecret, err := m.newTLSSecret(service, keyPair)
				if err != nil {
					return errors.Wrapf(err, "failed initailizing secret %s", service)
				}
				return m.client.Create(context.TODO(), tlsSecret)
			} else {
				return err
			}
		}
		return m.client.Update(context.TODO(), updateTLSSecret(secret, keyPair))
	})
}
