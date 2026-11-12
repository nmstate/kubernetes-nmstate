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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

// The test data below is opaque placeholder bytes; the digest does not parse
// certificates, so real key material is neither needed nor wanted here.
const (
	testLeafV1         = "leaf-certificate-v1"
	testLeafV2         = "leaf-certificate-v2"
	testIntermediateV1 = "intermediate-certificate-v1"
	testIntermediateV2 = "intermediate-certificate-v2"
	testKeyV1          = "placeholder-serving-key-v1"
	testKeyV2          = "placeholder-serving-key-v2"

	customPluginNamespace = "custom-plugin-namespace"
	customPluginName      = "custom-console-plugin"
	otherNamespace        = "some-other-namespace"

	servingCertServiceAnnotation = "service.alpha.openshift.io/serving-cert-secret-name"
)

// certChain builds a PEM-like certificate chain from the given blocks.
func certChain(blocks ...string) []byte {
	var b strings.Builder
	for _, block := range blocks {
		b.WriteString("-----BEGIN CERTIFICATE-----\n")
		b.WriteString(block)
		b.WriteString("\n-----END CERTIFICATE-----\n")
	}
	return []byte(b.String())
}

func servingCertSecret(namespace, name string, chain []byte, key string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       chain,
			corev1.TLSPrivateKeyKey: []byte(key),
		},
	}
}

// setEnvForTest sets (or unsets, for a nil value) an environment variable and
// restores its previous state when the current spec finishes.
func setEnvForTest(key string, value *string) {
	previous, wasSet := os.LookupEnv(key)
	DeferCleanup(func() {
		if wasSet {
			os.Setenv(key, previous)
		} else {
			os.Unsetenv(key)
		}
	})
	if value == nil {
		Expect(os.Unsetenv(key)).To(Succeed())
	} else {
		Expect(os.Setenv(key, *value)).To(Succeed())
	}
}

// copyUIPluginManifests copies the OpenShift console plugin templates to the
// location the operator image uses (see build/Dockerfile.operator).
func copyUIPluginManifests(manifestsDir string) error {
	srcDir := filepath.Join("..", "..", "deploy", "openshift", "ui-plugin")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	dstDir := filepath.Join(manifestsDir, "kubernetes-nmstate", "openshift", "ui-plugin") + string(filepath.Separator)
	for _, entry := range entries {
		if err := copyManifest(filepath.Join(srcDir, entry.Name()), dstDir); err != nil {
			return err
		}
	}
	return nil
}

// useUIPluginManifests points the operator at a temporary manifest directory
// holding the console plugin templates for the current spec.
func useUIPluginManifests() {
	manifestsDir := GinkgoT().TempDir()
	Expect(copyUIPluginManifests(manifestsDir)).To(Succeed())
	previous := names.ManifestDir
	names.ManifestDir = manifestsDir
	DeferCleanup(func() { names.ManifestDir = previous })
}

var _ = Describe("Console plugin serving certificate", func() {
	baseSecret := func() *corev1.Secret {
		return servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
			certChain(testLeafV1, testIntermediateV1), testKeyV1)
	}

	Context("digest", func() {
		It("is a pinned hex SHA-256 over the length-prefixed tls.crt and tls.key", func() {
			// Computed independently from the documented encoding. A change
			// here changes the annotation of every existing plugin
			// Deployment, so it rolls the plugin on upgrade.
			Expect(pluginServingCertHash(baseSecret())).To(Equal(
				"3222c018a3daec15e3f8f792d7ef585e5680c2b926f71b9ac68e2b493cadbddf"))
		})

		It("is stable for identical bytes regardless of metadata, type and other data keys", func() {
			other := baseSecret()
			other.Namespace = "elsewhere"
			other.Name = "another-name"
			other.ResourceVersion = "42"
			other.UID = types.UID("some-uid")
			other.Labels = map[string]string{"foo": "bar"}
			other.Annotations = map[string]string{
				"service.beta.openshift.io/expiry":                    "2030-01-01T00:00:00Z",
				"service.alpha.openshift.io/originating-service-name": "nmstate-console-plugin",
			}
			other.Type = corev1.SecretTypeOpaque
			other.Data["unrelated"] = []byte("ignored")
			Expect(pluginServingCertHash(other)).To(Equal(pluginServingCertHash(baseSecret())))
		})

		It("covers the complete certificate chain, not just the leaf", func() {
			digests := map[string]string{"base": pluginServingCertHash(baseSecret())}
			for name, chain := range map[string][]byte{
				"leaf rotated":         certChain(testLeafV2, testIntermediateV1),
				"intermediate rotated": certChain(testLeafV1, testIntermediateV2),
				"intermediate dropped": certChain(testLeafV1),
				"intermediate added":   certChain(testLeafV1, testIntermediateV1, testIntermediateV2),
				"chain reordered":      certChain(testIntermediateV1, testLeafV1),
			} {
				secret := baseSecret()
				secret.Data[corev1.TLSCertKey] = chain
				digests[name] = pluginServingCertHash(secret)
			}
			seen := map[string]string{}
			for name, digest := range digests {
				Expect(seen).ToNot(HaveKey(digest), "%q collides with %q", name, seen[digest])
				seen[digest] = name
			}
		})

		It("changes when only the private key changes", func() {
			rotated := baseSecret()
			rotated.Data[corev1.TLSPrivateKeyKey] = []byte(testKeyV2)
			Expect(pluginServingCertHash(rotated)).ToNot(Equal(pluginServingCertHash(baseSecret())))
		})

		It("is not ambiguous across the certificate/key boundary", func() {
			a := servingCertSecret("ns", "n", []byte("AB"), "C")
			b := servingCertSecret("ns", "n", []byte("A"), "BC")
			Expect(pluginServingCertHash(a)).ToNot(Equal(pluginServingCertHash(b)))

			swapped := servingCertSecret("ns", "n", []byte(testKeyV1), string(certChain(testLeafV1)))
			straight := servingCertSecret("ns", "n", certChain(testLeafV1), testKeyV1)
			Expect(pluginServingCertHash(swapped)).ToNot(Equal(pluginServingCertHash(straight)))
		})

		It("distinguishes a missing key from an empty value", func() {
			missingKey := &corev1.Secret{Data: map[string][]byte{corev1.TLSCertKey: certChain(testLeafV1)}}
			emptyKey := &corev1.Secret{Data: map[string][]byte{
				corev1.TLSCertKey:       certChain(testLeafV1),
				corev1.TLSPrivateKeyKey: {},
			}}
			Expect(pluginServingCertHash(missingKey)).ToNot(Equal(pluginServingCertHash(emptyKey)))
			Expect(pluginServingCertHash(&corev1.Secret{})).ToNot(Equal(pluginServingCertHash(&corev1.Secret{
				Data: map[string][]byte{corev1.TLSCertKey: {}, corev1.TLSPrivateKeyKey: {}},
			})))
		})
	})

	for _, tc := range []struct {
		description string
		envValue    *string
		namespace   string
		otherNs     string
	}{
		{"with HANDLER_NAMESPACE unset (default namespace)", nil, defaultPluginNamespace, otherNamespace},
		{"with a custom HANDLER_NAMESPACE", new(customPluginNamespace), customPluginNamespace, defaultPluginNamespace},
	} {
		Context(tc.description, func() {
			var (
				secret     *corev1.Secret
				pred       = pluginServingCertPredicate()
				reconciler *NMStateReconciler
			)
			BeforeEach(func() {
				setEnvForTest("HANDLER_NAMESPACE", tc.envValue)
				secret = servingCertSecret(tc.namespace, pluginServingCertSecretName,
					certChain(testLeafV1, testIntermediateV1), testKeyV1)
				cl := fake.NewClientBuilder().WithScheme(testScheme()).WithRuntimeObjects(
					&nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate"}},
				).Build()
				reconciler = &NMStateReconciler{Client: cl, Log: ctrl.Log.WithName("test")}
			})

			It("resolves the Secret in the effective plugin namespace", func() {
				Expect(pluginServingCertKey()).To(Equal(types.NamespacedName{
					Namespace: tc.namespace, Name: pluginServingCertSecretName,
				}))
			})

			It("restricts the cache to the Secret namespace and name", func() {
				byObject := PluginServingCertCacheByObject()
				Expect(byObject.Namespaces).To(HaveLen(1))
				Expect(byObject.Namespaces).To(HaveKey(tc.namespace))
				Expect(byObject.Label).To(BeNil())
				Expect(byObject.Field.String()).To(Equal("metadata.name=" + pluginServingCertSecretName))
				Expect(byObject.Field.Matches(fields.Set{"metadata.name": pluginServingCertSecretName})).To(BeTrue())
				Expect(byObject.Field.Matches(fields.Set{"metadata.name": "other-secret"})).To(BeFalse())
			})

			It("passes create, delete and generic events for the serving certificate Secret", func() {
				Expect(pred.Create(event.TypedCreateEvent[*corev1.Secret]{Object: secret})).To(BeTrue())
				Expect(pred.Delete(event.TypedDeleteEvent[*corev1.Secret]{Object: secret})).To(BeTrue())
				Expect(pred.Generic(event.TypedGenericEvent[*corev1.Secret]{Object: secret})).To(BeTrue())
			})

			It("drops events for other Secrets in the plugin namespace and for the same name elsewhere", func() {
				for _, other := range []*corev1.Secret{
					servingCertSecret(tc.namespace, "other-secret", certChain(testLeafV1), testKeyV1),
					servingCertSecret(tc.otherNs, pluginServingCertSecretName, certChain(testLeafV1), testKeyV1),
				} {
					rotated := other.DeepCopy()
					rotated.Data[corev1.TLSCertKey] = certChain(testLeafV2)
					Expect(pred.Create(event.TypedCreateEvent[*corev1.Secret]{Object: other})).To(BeFalse())
					Expect(pred.Update(event.TypedUpdateEvent[*corev1.Secret]{ObjectOld: other, ObjectNew: rotated})).To(BeFalse())
					Expect(pred.Delete(event.TypedDeleteEvent[*corev1.Secret]{Object: other})).To(BeFalse())
					Expect(pred.Generic(event.TypedGenericEvent[*corev1.Secret]{Object: other})).To(BeFalse())
					Expect(reconciler.nmstateRequestsFromPluginServingCert(context.Background(), other)).To(BeEmpty())
				}
			})

			It("passes updates that rotate the certificate chain or the key", func() {
				chainRotated := secret.DeepCopy()
				chainRotated.Data[corev1.TLSCertKey] = certChain(testLeafV1, testIntermediateV2)
				Expect(pred.Update(event.TypedUpdateEvent[*corev1.Secret]{ObjectOld: secret, ObjectNew: chainRotated})).To(BeTrue())

				keyRotated := secret.DeepCopy()
				keyRotated.Data[corev1.TLSPrivateKeyKey] = []byte(testKeyV2)
				Expect(pred.Update(event.TypedUpdateEvent[*corev1.Secret]{ObjectOld: secret, ObjectNew: keyRotated})).To(BeTrue())
			})

			It("passes updates without a previous object", func() {
				Expect(pred.Update(event.TypedUpdateEvent[*corev1.Secret]{ObjectNew: secret})).To(BeTrue())
			})

			It("drops metadata-only updates", func() {
				updated := secret.DeepCopy()
				updated.ResourceVersion = "2"
				updated.Labels = map[string]string{"foo": "bar"}
				updated.Annotations = map[string]string{"service.beta.openshift.io/expiry": "2030-01-01T00:00:00Z"}
				Expect(pred.Update(event.TypedUpdateEvent[*corev1.Secret]{ObjectOld: secret, ObjectNew: updated})).To(BeFalse())
			})

			It("maps the serving certificate Secret to every NMState CR without relying on owner references", func() {
				Expect(secret.OwnerReferences).To(BeEmpty())
				Expect(reconciler.nmstateRequestsFromPluginServingCert(context.Background(), secret)).To(ConsistOf(
					ctrl.Request{NamespacedName: types.NamespacedName{Name: "nmstate"}},
				))

				Expect(reconciler.Client.Create(context.Background(),
					&nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate-two"}})).To(Succeed())
				Expect(reconciler.nmstateRequestsFromPluginServingCert(context.Background(), secret)).To(ConsistOf(
					ctrl.Request{NamespacedName: types.NamespacedName{Name: "nmstate"}},
					ctrl.Request{NamespacedName: types.NamespacedName{Name: "nmstate-two"}},
				))
			})
		})
	}
})

var _ = Describe("Console plugin rollout on serving certificate rotation", func() {
	var (
		ctx        = context.Background()
		cl         client.Client
		instance   *nmstatev1.NMState
		reconciler *NMStateReconciler
	)

	newReconciler := func() *NMStateReconciler {
		return &NMStateReconciler{
			Client:      cl,
			APIClient:   cl,
			Scheme:      scheme.Scheme,
			Log:         ctrl.Log.WithName("controllers").WithName("NMState"),
			IsOpenShift: true,
		}
	}

	getPluginDeployment := func(namespace, name string) *appsv1.Deployment {
		deployment := &appsv1.Deployment{}
		Expect(cl.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, deployment)).To(Succeed())
		return deployment
	}
	getDeployment := func() *appsv1.Deployment {
		return getPluginDeployment(defaultPluginNamespace, defaultPluginName)
	}

	digestOf := func(deployment *appsv1.Deployment) (string, bool) {
		digest, ok := deployment.Spec.Template.Annotations[pluginServingCertHashAnnotation]
		return digest, ok
	}

	expectAnnotatedService := func(namespace string) {
		service := &corev1.Service{}
		Expect(cl.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaultPluginName}, service)).To(Succeed())
		Expect(service.Annotations).To(HaveKeyWithValue(servingCertServiceAnnotation, pluginServingCertSecretName))
	}

	apply := func() {
		Expect(reconciler.applyOpenshiftUIPlugin(ctx, instance)).To(Succeed())
	}

	BeforeEach(func() {
		useUIPluginManifests()
		setEnvForTest("PLUGIN_NAME", nil)
		setEnvForTest("HANDLER_NAMESPACE", nil)

		instance = &nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate", UID: "nmstate-uid"}}
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme()).WithRuntimeObjects(instance).Build()
		cl = &applyCapableFakeClient{Client: fakeClient}
		reconciler = newReconciler()
	})

	It("creates the annotated Service on first install while the Secret is absent, then rolls out once it appears", func() {
		apply()

		expectAnnotatedService(defaultPluginNamespace)
		deployment := getDeployment()
		_, hasDigest := digestOf(deployment)
		Expect(hasDigest).To(BeFalse(), "no digest can be rendered before service-ca issues the certificate")
		Expect(deployment.Spec.Template.Annotations).To(HaveKey("target.workload.openshift.io/management"))
		Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(SatisfyAll(
			HaveField("Name", "plugin-serving-cert"),
			HaveField("VolumeSource.Secret.SecretName", pluginServingCertSecretName),
		)))

		By("reconciling again while the Secret is still absent")
		apply()
		_, hasDigest = digestOf(getDeployment())
		Expect(hasDigest).To(BeFalse())

		By("service-ca creating the Secret")
		secret := servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
			certChain(testLeafV1, testIntermediateV1), testKeyV1)
		Expect(cl.Create(ctx, secret)).To(Succeed())
		apply()

		digest, hasDigest := digestOf(getDeployment())
		Expect(hasDigest).To(BeTrue())
		Expect(digest).To(Equal(pluginServingCertHash(secret)))
	})

	Context("when the serving certificate Secret exists", func() {
		var secret *corev1.Secret
		BeforeEach(func() {
			secret = servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
				certChain(testLeafV1, testIntermediateV1), testKeyV1)
			Expect(cl.Create(ctx, secret)).To(Succeed())
			apply()
		})

		rotate := func(chain []byte, key string) {
			Expect(cl.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
			secret.Data[corev1.TLSCertKey] = chain
			secret.Data[corev1.TLSPrivateKeyKey] = []byte(key)
			Expect(cl.Update(ctx, secret)).To(Succeed())
		}

		It("places only the digest on the Deployment pod template", func() {
			deployment := getDeployment()
			digest, _ := digestOf(deployment)
			Expect(digest).To(Equal(pluginServingCertHash(secret)))
			Expect(digest).To(MatchRegexp("^[0-9a-f]{64}$"))
			Expect(deployment.Annotations).ToNot(HaveKey(pluginServingCertHashAnnotation),
				"the digest belongs on the pod template, not the Deployment metadata")

			serialized, err := json.Marshal(deployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(serialized)).ToNot(ContainSubstring(testKeyV1))
			Expect(string(serialized)).ToNot(ContainSubstring(testLeafV1))
		})

		It("changes the pod template when service-ca rotates the certificate chain and key", func() {
			before, _ := digestOf(getDeployment())
			rotate(certChain(testLeafV2, testIntermediateV2), testKeyV2)
			apply()

			after, _ := digestOf(getDeployment())
			Expect(after).ToNot(Equal(before))
			Expect(after).To(Equal(pluginServingCertHash(secret)))
		})

		It("keeps the pod template unchanged on repeated reconciles and metadata-only Secret updates", func() {
			template := getDeployment().Spec.Template

			apply()
			Expect(apiequality.Semantic.DeepEqual(getDeployment().Spec.Template, template)).To(BeTrue())

			Expect(cl.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
			secret.Labels = map[string]string{"foo": "bar"}
			secret.Annotations = map[string]string{"service.beta.openshift.io/expiry": "2030-01-01T00:00:00Z"}
			Expect(cl.Update(ctx, secret)).To(Succeed())
			apply()
			Expect(apiequality.Semantic.DeepEqual(getDeployment().Spec.Template, template)).To(BeTrue())
		})

		It("keeps the current digest while the Secret is deleted and rolls out the recreated certificate", func() {
			before, _ := digestOf(getDeployment())

			Expect(cl.Delete(ctx, secret)).To(Succeed())
			apply()
			during, hasDigest := digestOf(getDeployment())
			Expect(hasDigest).To(BeTrue())
			Expect(during).To(Equal(before), "a missing Secret must not roll out pods that cannot mount it")
			expectAnnotatedService(defaultPluginNamespace)

			recreated := servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
				certChain(testLeafV2, testIntermediateV1), testKeyV2)
			Expect(cl.Create(ctx, recreated)).To(Succeed())
			apply()
			after, _ := digestOf(getDeployment())
			Expect(after).To(Equal(pluginServingCertHash(recreated)))
			Expect(after).ToNot(Equal(before))
		})

		It("keeps the digest of the live Deployment, not of a lagging cache, while the Secret is missing", func() {
			live, _ := digestOf(getDeployment())
			stale := strings.Repeat("0", 64)
			underlying, ok := cl.(*applyCapableFakeClient).Client.(client.WithWatch)
			Expect(ok).To(BeTrue())
			// The cached client still reports the previous digest.
			reconciler.Client = &applyCapableFakeClient{Client: interceptor.NewClient(underlying, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if err := c.Get(ctx, key, obj, opts...); err != nil {
						return err
					}
					if deployment, ok := obj.(*appsv1.Deployment); ok && deployment.Spec.Template.Annotations != nil {
						deployment.Spec.Template.Annotations[pluginServingCertHashAnnotation] = stale
					}
					return nil
				},
			})}

			Expect(cl.Delete(ctx, secret)).To(Succeed())
			apply()
			kept, _ := digestOf(getDeployment())
			Expect(kept).To(Equal(live))
		})

		It("never renders an edited annotation value that is not a digest while the Secret is missing", func() {
			// Closes the annotation, injects a ClusterRoleBinding, and
			// opens a Deployment whose pod template annotations swallow
			// the rest of the rendered plugin Deployment, so the result
			// parses if the value is rendered as is.
			injected := "x\"\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding\n" +
				"metadata:\n  name: injected\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n" +
				"  name: cluster-admin\nsubjects:\n- kind: ServiceAccount\n  name: attacker\n  namespace: default\n---\n" +
				"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: injected\n  namespace: default\nspec:\n" +
				"  selector:\n    matchLabels:\n      app: injected\n  template:\n    metadata:\n      annotations:\n        a: \""
			digest := pluginServingCertHash(secret)
			Expect(cl.Delete(ctx, secret)).To(Succeed())

			for _, edited := range []string{injected, `not-a-digest"`, strings.ToUpper(digest), digest[1:], digest + "0", digest + "\n"} {
				deployment := getDeployment()
				deployment.Spec.Template.Annotations[pluginServingCertHashAnnotation] = edited
				Expect(cl.Update(ctx, deployment)).To(Succeed())

				apply()
				_, hasDigest := digestOf(getDeployment())
				Expect(hasDigest).To(BeFalse(), "edited value %q must not be rendered", edited)
				bindings := &rbacv1.ClusterRoleBindingList{}
				Expect(cl.List(ctx, bindings)).To(Succeed())
				Expect(bindings.Items).To(BeEmpty())
				err := cl.Get(ctx, types.NamespacedName{Namespace: "default", Name: "injected"}, &appsv1.Deployment{})
				Expect(err).To(Satisfy(apierrors.IsNotFound))
			}
		})

		Context("after an operator restart", func() {
			// On start the Secret informer lists the existing Secret
			// and delivers a Create event. A fresh reconciler sees only
			// what is stored in the cluster.
			restartAndReconcile := func() {
				stored := &corev1.Secret{}
				Expect(cl.Get(ctx, client.ObjectKeyFromObject(secret), stored)).To(Succeed())
				restarted := newReconciler()
				Expect(pluginServingCertPredicate().Create(event.TypedCreateEvent[*corev1.Secret]{Object: stored})).To(BeTrue())
				Expect(restarted.nmstateRequestsFromPluginServingCert(ctx, stored)).To(ConsistOf(
					ctrl.Request{NamespacedName: types.NamespacedName{Name: instance.Name}},
				))
				Expect(restarted.applyOpenshiftUIPlugin(ctx, instance)).To(Succeed())
			}

			It("rolls out a rotation that happened while the operator was not running", func() {
				before, _ := digestOf(getDeployment())
				rotate(certChain(testLeafV2, testIntermediateV1), testKeyV1)

				restartAndReconcile()

				after, _ := digestOf(getDeployment())
				Expect(after).To(Equal(pluginServingCertHash(secret)))
				Expect(after).ToNot(Equal(before))
			})

			It("does not change the pod template when the certificate is unchanged", func() {
				template := getDeployment().Spec.Template
				restartAndReconcile()
				Expect(apiequality.Semantic.DeepEqual(getDeployment().Spec.Template, template)).To(BeTrue())
			})
		})
	})

	It("uses the custom plugin namespace for render, read and digest", func() {
		setEnvForTest("HANDLER_NAMESPACE", new(customPluginNamespace))

		// Same name in the default namespace must be ignored.
		Expect(cl.Create(ctx, servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
			certChain(testLeafV2), testKeyV2))).To(Succeed())
		apply()
		_, hasDigest := digestOf(getPluginDeployment(customPluginNamespace, defaultPluginName))
		Expect(hasDigest).To(BeFalse())

		secret := servingCertSecret(customPluginNamespace, pluginServingCertSecretName,
			certChain(testLeafV1, testIntermediateV1), testKeyV1)
		Expect(cl.Create(ctx, secret)).To(Succeed())
		apply()

		digest, _ := digestOf(getPluginDeployment(customPluginNamespace, defaultPluginName))
		Expect(digest).To(Equal(pluginServingCertHash(secret)))
		expectAnnotatedService(customPluginNamespace)
	})

	It("keeps the digest of the Deployment named by PLUGIN_NAME while the Secret is missing", func() {
		setEnvForTest("PLUGIN_NAME", new(customPluginName))
		secret := servingCertSecret(defaultPluginNamespace, pluginServingCertSecretName,
			certChain(testLeafV1, testIntermediateV1), testKeyV1)
		Expect(cl.Create(ctx, secret)).To(Succeed())
		apply()
		before, _ := digestOf(getPluginDeployment(defaultPluginNamespace, customPluginName))
		Expect(before).To(Equal(pluginServingCertHash(secret)))

		Expect(cl.Delete(ctx, secret)).To(Succeed())
		apply()
		during, _ := digestOf(getPluginDeployment(defaultPluginNamespace, customPluginName))
		Expect(during).To(Equal(before))
	})

	Context("when reading the cluster state fails", func() {
		readError := apierrors.NewServiceUnavailable("injected read failure")

		useFailingClient := func(failOn schema.GroupKind) {
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(instance).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						gvk, err := c.GroupVersionKindFor(obj)
						if err != nil {
							return err
						}
						if gvk.GroupKind() == failOn {
							return readError
						}
						return c.Get(ctx, key, obj, opts...)
					},
				}).Build()
			cl = &applyCapableFakeClient{Client: fakeClient}
			reconciler = newReconciler()
		}

		expectNothingApplied := func() {
			err := cl.Get(ctx, types.NamespacedName{Namespace: defaultPluginNamespace, Name: defaultPluginName}, &corev1.Service{})
			Expect(err).To(Satisfy(apierrors.IsNotFound))
		}

		It("returns the Secret read error without applying anything", func() {
			useFailingClient(schema.GroupKind{Kind: "Secret"})
			err := reconciler.applyOpenshiftUIPlugin(ctx, instance)
			Expect(err).To(MatchError(readError))
			Expect(err.Error()).To(ContainSubstring(defaultPluginNamespace + "/" + pluginServingCertSecretName))
			expectNothingApplied()
		})

		It("returns the Deployment read error when the Secret is missing, without applying anything", func() {
			useFailingClient(schema.GroupKind{Group: "apps", Kind: "Deployment"})
			err := reconciler.applyOpenshiftUIPlugin(ctx, instance)
			Expect(err).To(MatchError(readError))
			Expect(err.Error()).To(ContainSubstring(defaultPluginNamespace + "/" + defaultPluginName))
			expectNothingApplied()
		})
	})
})
