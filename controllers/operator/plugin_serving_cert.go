/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
)

const (
	// pluginServingCertSecretName is the Secret that the OpenShift service-ca
	// operator populates with the console plugin serving certificate, as
	// requested by the annotation on the plugin Service. The same name is
	// rendered into the Service annotation and the Deployment volume.
	pluginServingCertSecretName = "plugin-serving-cert"

	// pluginServingCertHashAnnotation is set on the console plugin pod
	// template. Its value is a digest of the serving certificate and key, so
	// a certificate rotation changes the pod template and rolls the
	// Deployment. nginx only loads the certificate at startup.
	pluginServingCertHashAnnotation = "nmstate.io/plugin-serving-cert-hash"

	defaultPluginNamespace = "openshift-nmstate"
	defaultPluginName      = "nmstate-console-plugin"
)

// pluginServingCertHashPattern matches the hex SHA-256 digests produced by
// pluginServingCertHash.
var pluginServingCertHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// pluginServingCertHashedKeys are the Secret data keys mounted into the
// plugin pod and loaded by nginx, in the fixed order they are hashed.
var pluginServingCertHashedKeys = []string{corev1.TLSCertKey, corev1.TLSPrivateKeyKey}

// pluginNamespace returns the namespace the console plugin is deployed to.
// Every path that renders, reads, watches or caches plugin objects must use
// it so they agree on the same default.
func pluginNamespace() string {
	return environment.GetEnvVar("HANDLER_NAMESPACE", defaultPluginNamespace)
}

// pluginName returns the name of the console plugin and of its Deployment.
func pluginName() string {
	return environment.GetEnvVar("PLUGIN_NAME", defaultPluginName)
}

// pluginServingCertKey returns the namespaced name of the console plugin
// serving certificate Secret.
func pluginServingCertKey() types.NamespacedName {
	return types.NamespacedName{Namespace: pluginNamespace(), Name: pluginServingCertSecretName}
}

// PluginServingCertCacheByObject restricts the manager cache for Secrets to
// the console plugin serving certificate Secret, by namespace and by name.
// This Secret is the only one the operator reads or watches through the
// cache, so the operator never lists or caches other Secrets. The informer
// is created lazily, so it only starts on OpenShift, where the Secret is
// watched.
func PluginServingCertCacheByObject() cache.ByObject {
	key := pluginServingCertKey()
	return cache.ByObject{
		Namespaces: map[string]cache.Config{key.Namespace: {}},
		Field:      fields.OneTermEqualSelector("metadata.name", key.Name),
	}
}

// pluginServingCertHash returns the hex SHA-256 digest of the serving
// certificate chain and private key stored in secret.
//
// Each key in pluginServingCertHashedKeys is written in a fixed order as:
// the key name length and the name, then a presence flag, then the value
// length and the value. Lengths are big-endian uint64. Because every field
// has an explicit length, bytes can't move between fields (for example the
// end of the chain and the start of the key) without changing the digest.
// A missing key also hashes differently from an empty value. Other data
// keys and all metadata are ignored, so metadata-only updates keep the
// digest stable.
func pluginServingCertHash(secret *corev1.Secret) string {
	h := sha256.New()
	for _, key := range pluginServingCertHashedKeys {
		writeLengthPrefixed(h, []byte(key))
		value, present := secret.Data[key]
		if present {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		writeLengthPrefixed(h, value)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeLengthPrefixed(h hash.Hash, b []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(b)))
	// hash.Hash.Write never returns an error.
	h.Write(length[:])
	h.Write(b)
}

// pluginServingCertHashForRender returns the digest to render on the console
// plugin pod template.
//
// When the Secret exists, the digest is computed from its current content.
//
// When the Secret is missing, rendering must still continue. On first
// install the annotated Service has to be created so that service-ca can
// issue the certificate. In that case a valid digest already on the live
// Deployment is kept, if there is one. A transient deletion then doesn't
// roll out pods that can't mount the Secret yet. The watch on the Secret
// rolls out the new digest once service-ca creates the Secret again. An
// empty string means no digest is rendered.
func (r *NMStateReconciler) pluginServingCertHashForRender(ctx context.Context) (string, error) {
	key := pluginServingCertKey()
	secret := &corev1.Secret{}
	err := r.Get(ctx, key, secret)
	switch {
	case err == nil:
		return pluginServingCertHash(secret), nil
	case !apierrors.IsNotFound(err):
		return "", fmt.Errorf("failed getting console plugin serving certificate secret %s: %w", key, err)
	}

	// Read the live Deployment rather than the cache. Right after a
	// rotation the cache may still hold the previous digest, and keeping
	// that one would roll back to the old certificate.
	deployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{Namespace: key.Namespace, Name: pluginName()}
	err = r.APIClient.Get(ctx, deploymentKey, deployment)
	switch {
	case apierrors.IsNotFound(err):
		r.Log.Info("Console plugin serving certificate secret not found yet; rendering the plugin without a certificate digest",
			"secret", key.String())
		return "", nil
	case err != nil:
		return "", fmt.Errorf("failed getting console plugin deployment %s: %w", deploymentKey, err)
	}
	// Anyone who can patch the Deployment can edit this annotation, and the
	// value is rendered into the manifest template. Only keep a value of the
	// exact form pluginServingCertHash produces, never anything else.
	digest := deployment.Spec.Template.Annotations[pluginServingCertHashAnnotation]
	if !isPluginServingCertHash(digest) {
		r.Log.Info("Console plugin serving certificate secret not found and the current deployment has no valid certificate digest; "+
			"rendering the plugin without a certificate digest", "secret", key.String(), "deployment", deploymentKey.String())
		return "", nil
	}
	r.Log.Info("Console plugin serving certificate secret not found; keeping the certificate digest of the current deployment",
		"secret", key.String())
	return digest, nil
}

// isPluginServingCertHash reports whether s has the form of a digest
// returned by pluginServingCertHash: 64 lowercase hex characters.
func isPluginServingCertHash(s string) bool {
	return pluginServingCertHashPattern.MatchString(s)
}

// isPluginServingCert reports whether secret is the console plugin serving
// certificate Secret in the effective plugin namespace.
func isPluginServingCert(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}
	key := pluginServingCertKey()
	return secret.Namespace == key.Namespace && secret.Name == key.Name
}

// pluginServingCertPredicate passes events for the console plugin serving
// certificate Secret only. Updates pass only when the certificate or key
// bytes change. Metadata-only updates, which don't change the digest, are
// dropped.
func pluginServingCertPredicate() predicate.TypedPredicate[*corev1.Secret] {
	return predicate.TypedFuncs[*corev1.Secret]{
		CreateFunc: func(e event.TypedCreateEvent[*corev1.Secret]) bool {
			return isPluginServingCert(e.Object)
		},
		UpdateFunc: func(e event.TypedUpdateEvent[*corev1.Secret]) bool {
			if !isPluginServingCert(e.ObjectNew) {
				return false
			}
			if e.ObjectOld == nil {
				return true
			}
			return pluginServingCertHash(e.ObjectOld) != pluginServingCertHash(e.ObjectNew)
		},
		DeleteFunc: func(e event.TypedDeleteEvent[*corev1.Secret]) bool {
			return isPluginServingCert(e.Object)
		},
		GenericFunc: func(e event.TypedGenericEvent[*corev1.Secret]) bool {
			return isPluginServingCert(e.Object)
		},
	}
}

// nmstateRequestsFromPluginServingCert maps events on the console plugin
// serving certificate Secret to reconcile requests for the NMState CRs, so
// that the plugin pod template picks up the new certificate digest. The
// Secret is created by service-ca and has no owner reference to NMState, so
// the mapping is explicit.
func (r *NMStateReconciler) nmstateRequestsFromPluginServingCert(ctx context.Context, secret *corev1.Secret) []ctrl.Request {
	if !isPluginServingCert(secret) {
		return nil
	}
	return r.nmstateRequests(ctx, "console plugin serving certificate secret")
}
