package certificate

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate/triple"
)

func tlsSecret(service types.NamespacedName, keyPair *triple.KeyPair) corev1.Secret {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       triple.EncodeCertPEM(keyPair.Cert),
			corev1.TLSPrivateKeyKey: triple.EncodePrivateKeyPEM(keyPair.Key),
		},
		Type: corev1.SecretTypeTLS,
	}
	return secret
}

func (m *Manager) applyTLSSecret(service types.NamespacedName, keyPair *triple.KeyPair) error {
	tlsSecret := tlsSecret(service, keyPair)

	err := m.get(service, &corev1.Secret{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return m.client.Create(context.TODO(), &tlsSecret)
		} else {
			return err
		}
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return m.client.Update(context.TODO(), &tlsSecret)
	})
}
