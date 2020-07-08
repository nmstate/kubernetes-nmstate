package certificate

import (
	"context"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate/triple"
)

const (
	secretManagedAnnotatoinKey = "kubevirt.io/kube-admission-webhook"
)

func updateTLSSecret(secret corev1.Secret, keyPair *triple.KeyPair) *corev1.Secret {
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[secretManagedAnnotatoinKey] = ""
	secret.Data = map[string][]byte{
		corev1.TLSCertKey:       triple.EncodeCertPEM(keyPair.Cert),
		corev1.TLSPrivateKeyKey: triple.EncodePrivateKeyPEM(keyPair.Key),
	}
	secret.Type = corev1.SecretTypeTLS
	return &secret
}

func (m *Manager) setSecretOwnership(secretKey types.NamespacedName, secret *corev1.Secret) error {

	service := corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:      secretKey.Name,
		Namespace: secretKey.Namespace,
	}}

	err := m.get(secretKey, &service)
	if err != nil {
		if apierrors.IsNotFound(err) {
			m.log.Info("Orphan secret, service is not found")
			return nil
		}
		return errors.Wrapf(err, "failed getting service %s to set secret owner", secretKey)
	}

	serviceGVK, err := apiutil.GVKForObject(&service, scheme.Scheme)
	if err != nil {
		return errors.Wrapf(err, "failed getting gvk from service %s", secretKey)
	}

	secret.OwnerReferences = []metav1.OwnerReference{
		{
			Name:       service.Name,
			Kind:       serviceGVK.Kind,
			APIVersion: serviceGVK.GroupVersion().String(),
			UID:        service.UID,
		}}

	return nil
}

func (m *Manager) newTLSSecret(secretKey types.NamespacedName, keyPair *triple.KeyPair) (*corev1.Secret, error) {

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretKey.Name,
			Namespace:   secretKey.Namespace,
			Annotations: map[string]string{},
		},
	}

	err := m.setSecretOwnership(secretKey, &secret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed setting ownership to secret %s", secretKey)
	}

	return &secret, nil
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
				return m.client.Create(context.TODO(), updateTLSSecret(*tlsSecret, keyPair))
			} else {
				return err
			}
		}
		return m.client.Update(context.TODO(), updateTLSSecret(secret, keyPair))
	})
}

// verifyTLSSecret will verify that the caBundle and Secret are valid and can
// be used to verify
func (m *Manager) verifyTLSSecret(secretKey types.NamespacedName, caBundle []byte) error {
	secret := corev1.Secret{}
	err := m.get(secretKey, &secret)
	if err != nil {
		return errors.Wrapf(err, "failed getting TLS secret %s", secretKey)
	}

	keyPEM, found := secret.Data[corev1.TLSPrivateKeyKey]
	if !found {
		return errors.New("TLS key not found")
	}

	certsPEM, found := secret.Data[corev1.TLSCertKey]
	if !found {
		return errors.New("TLS certs not found")
	}

	err = triple.VerifyTLS(certsPEM, keyPEM, []byte(caBundle))
	if err != nil {
		return errors.Wrapf(err, "failed verifying TLS from server Secret %s", secretKey)
	}

	return nil
}
