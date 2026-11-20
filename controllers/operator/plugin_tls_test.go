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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/nmstate/kubernetes-nmstate/api/names"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Console plugin TLS policy", func() {
	It("renders API groups and rolls out a group-only change", func() { testPluginTLSGroupChange(GinkgoTB()) })
	It("pins configuration through a failed Deployment update and rollback", func() { testPluginTLSFailedDeploymentRollback(GinkgoTB()) })
	It("migrates the legacy configuration without changing content used by existing pods", func() {
		t := GinkgoTB()
		r, instance, cl, _ := setupPluginTLS(t, map[string]any{"type": "Modern"})
		legacy := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: pluginNginxConfigName, Namespace: "nmstate"},
			Data:       map[string]string{"nginx.conf": "legacy nginx configuration"},
		}
		if err := cl.Create(context.Background(), legacy); err != nil {
			t.Fatal(err)
		}
		if err := r.applyManifests(instance, context.Background()); err != nil {
			t.Fatal(err)
		}
		current, _ := pluginTLSObjects(t, cl)
		if current.Name == legacy.Name || current.Immutable == nil || !*current.Immutable {
			t.Fatal("plugin was not migrated to an immutable configuration revision")
		}
		owner := metav1.GetControllerOf(current)
		if owner == nil || owner.UID != instance.UID || owner.Name != instance.Name {
			t.Fatal("configuration revision does not retain NMState ownership for garbage collection")
		}
		stored := &corev1.ConfigMap{}
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(legacy), stored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored.Data, legacy.Data) || stored.Immutable != nil {
			t.Fatal("legacy configuration was modified during migration")
		}
	})
	It("renders cluster profiles and binds the plugin template to the config", func() { testPluginTLSProfilesRendered(GinkgoTB()) })
	It("rolls out changed configuration only after a successful config write", func() { testPluginTLSConfigRollout(GinkgoTB()) })
	It("reports partial profile support", func() { testPluginTLSPartialProfileReportsWarning(GinkgoTB()) })
	It("does not apply a plugin profile with incompatible protocol and ciphers", func() {
		testPluginTLSInvalidProfileDoesNotApplyPlugin(GinkgoTB())
	})
})

func newPluginTLSScheme(t testing.TB) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme, nmstatev1.AddToScheme, nmstatev1beta1.AddToScheme, apiextv1.AddToScheme,
	} {
		if err := add(s); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// The fake client cannot perform server-side apply. Capture the actual rendered
// resources via Create/Update, retaining resource versions across reconciles.
type pluginTLSClient struct {
	client.Client
	failConfig     bool
	failDeployment bool
	profileReads   int
}

func (c *pluginTLSClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if obj.GetObjectKind().GroupVersionKind().Kind == "APIServer" {
		c.profileReads++
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *pluginTLSClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if patch.Type() != types.ApplyPatchType {
		return c.Client.Patch(ctx, obj, patch, opts...)
	}
	if c.failConfig && obj.GetObjectKind().GroupVersionKind().Kind == "ConfigMap" &&
		strings.HasPrefix(obj.GetName(), pluginNginxConfigName) {
		return fmt.Errorf("injected plugin configuration failure")
	}
	if c.failDeployment && obj.GetObjectKind().GroupVersionKind().Kind == "Deployment" && obj.GetName() == "nmstate-console-plugin" {
		return fmt.Errorf("injected plugin deployment failure")
	}
	if err := c.Create(ctx, obj); !apierrors.IsAlreadyExists(err) {
		return err
	}
	stored := obj.DeepCopyObject().(client.Object)
	if err := c.Client.Get(ctx, client.ObjectKeyFromObject(obj), stored); err != nil {
		return err
	}
	obj.SetResourceVersion(stored.GetResourceVersion())
	return c.Update(ctx, obj)
}

func testPluginTLSGroupChange(t testing.TB) {
	r, instance, cl, _ := setupPluginTLS(t, map[string]any{
		"type": "Custom",
		"custom": map[string]any{
			"minTLSVersion": "VersionTLS13",
			"ciphers":       []any{"TLS_AES_128_GCM_SHA256"},
			"groups":        []any{"secp256r1"},
		},
	})
	ctx := context.Background()
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	beforeConfig, beforeDeployment := pluginTLSObjects(t, cl)
	if !strings.Contains(beforeConfig.Data["nginx.conf"], "ssl_ecdh_curve      prime256v1;") {
		t.Fatal("API custom.groups restriction was not rendered")
	}
	api := &unstructured.Unstructured{}
	api.SetAPIVersion("config.openshift.io/v1")
	api.SetKind("APIServer")
	if err := cl.Get(ctx, client.ObjectKey{Name: "cluster"}, api); err != nil {
		t.Fatal(err)
	}
	if err := unstructured.SetNestedStringSlice(api.Object, []string{"X25519"}, "spec", "tlsSecurityProfile", "custom", "groups"); err != nil {
		t.Fatal(err)
	}
	if err := cl.Update(ctx, api); err != nil {
		t.Fatal(err)
	}
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	afterConfig, afterDeployment := pluginTLSObjects(t, cl)
	if !strings.Contains(afterConfig.Data["nginx.conf"], "ssl_ecdh_curve      X25519;") ||
		reflect.DeepEqual(beforeDeployment.Spec.Template, afterDeployment.Spec.Template) {
		t.Fatal("group-only update did not change the plugin configuration and template")
	}
}

func testPluginTLSFailedDeploymentRollback(t testing.TB) {
	r, instance, cl, _ := setupPluginTLS(t, map[string]any{"type": "Intermediate"})
	ctx := context.Background()
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	configA, deploymentA := pluginTLSObjects(t, cl)
	api := &unstructured.Unstructured{}
	api.SetAPIVersion("config.openshift.io/v1")
	api.SetKind("APIServer")
	if err := cl.Get(ctx, client.ObjectKey{Name: "cluster"}, api); err != nil {
		t.Fatal(err)
	}
	setProfile := func(name string) {
		t.Helper()
		if err := unstructured.SetNestedField(api.Object, name, "spec", "tlsSecurityProfile", "type"); err != nil {
			t.Fatal(err)
		}
		if err := cl.Update(ctx, api); err != nil {
			t.Fatal(err)
		}
	}
	setProfile("Modern")
	cl.failDeployment = true
	err := r.applyManifests(instance, ctx)
	if err == nil || !strings.Contains(err.Error(), "injected plugin deployment failure") {
		t.Fatalf("did not reach injected deployment failure: %v", err)
	}
	stillA, templateAfterFailure := pluginTLSObjects(t, cl)
	if !reflect.DeepEqual(templateAfterFailure.Spec.Template, deploymentA.Spec.Template) {
		t.Fatal("failed Deployment publication changed its template")
	}
	// A replacement created by the old ReplicaSet would use exactly this
	// template. Its referenced ConfigMap must still hold A, not the pending B.
	if !reflect.DeepEqual(stillA.Data, configA.Data) {
		t.Fatal("replacement under template A would mount configuration B")
	}
	if stillA.Immutable == nil || !*stillA.Immutable {
		t.Fatal("pinned configuration must be immutable")
	}
	setProfile("Intermediate")
	cl.failDeployment = false
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	rolledBack, templateAfterRollback := pluginTLSObjects(t, cl)
	if !reflect.DeepEqual(rolledBack.Data, configA.Data) ||
		!reflect.DeepEqual(templateAfterRollback.Spec.Template, deploymentA.Spec.Template) {
		t.Fatal("rollback did not restore the original configuration identity")
	}
	setProfile("Modern")
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	configB, deploymentB := pluginTLSObjects(t, cl)
	if configB.Name == configA.Name || reflect.DeepEqual(deploymentB.Spec.Template, deploymentA.Spec.Template) {
		t.Fatal("successful publication did not reference a new configuration revision")
	}
	preservedA := &corev1.ConfigMap{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(configA), preservedA); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preservedA.Data, configA.Data) {
		t.Fatal("publishing B mutated A's pinned configuration")
	}
}

func setupPluginTLS(t testing.TB, profile map[string]any) (*NMStateReconciler, *nmstatev1.NMState, *pluginTLSClient, string) {
	t.Helper()
	dir := t.TempDir()
	oldDir := names.ManifestDir
	names.ManifestDir = dir
	t.Cleanup(func() { names.ManifestDir = oldDir })
	if err := copyManifests(dir); err != nil {
		t.Fatal(err)
	}
	if err := copyManifest("../../deploy/handler/tls_config.yaml", filepath.Join(dir, "kubernetes-nmstate/handler")); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("../../deploy/openshift/ui-plugin/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if err := copyManifest(file, filepath.Join(dir, "kubernetes-nmstate/openshift/ui-plugin")); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range map[string]string{
		"HANDLER_NAMESPACE": "nmstate", "OPERATOR_NAMESPACE": "nmstate", "HANDLER_PREFIX": "",
		"RELATED_IMAGE_HANDLER_IMAGE": "example.invalid/handler:test", "HANDLER_IMAGE_PULL_POLICY": "IfNotPresent",
		"PLUGIN_NAME": "nmstate-console-plugin", "PLUGIN_IMAGE": "example.invalid/plugin:test", "PLUGIN_PORT": "9443",
		"MONITORING_NAMESPACE": "monitoring",
	} {
		t.Setenv(key, value)
	}
	api := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "config.openshift.io/v1", "kind": "APIServer",
		"metadata": map[string]any{"name": "cluster"}, "spec": map[string]any{},
	}}
	if profile != nil {
		api.Object["spec"] = map[string]any{"tlsSecurityProfile": profile}
	}
	console := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "operator.openshift.io/v1", "kind": "Console",
		"metadata": map[string]any{"name": "cluster"}, "spec": map[string]any{},
	}}
	instance := &nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate", UID: "test"}}
	// Reuse the suite's registered Kubernetes, nmstate and CRD types.
	s := newPluginTLSScheme(t)
	cl := &pluginTLSClient{Client: fake.NewClientBuilder().WithScheme(s).WithObjects(instance, api, console).Build()}
	return &NMStateReconciler{Client: cl, APIClient: cl, Scheme: s, IsOpenShift: true, Log: logr.Discard()}, instance, cl, dir
}

func pluginTLSObjects(t testing.TB, cl client.Client) (*corev1.ConfigMap, *appsv1.Deployment) {
	t.Helper()
	cm, deployment := &corev1.ConfigMap{}, &appsv1.Deployment{}
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "nmstate", Name: "nmstate-console-plugin"}, deployment); err != nil {
		t.Fatal(err)
	}
	var configName string
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == pluginNginxConfigName && volume.ConfigMap != nil {
			configName = volume.ConfigMap.Name
		}
	}
	if configName == "" {
		t.Fatal("plugin template does not reference nginx configuration")
	}
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "nmstate", Name: configName}, cm); err != nil {
		t.Fatal(err)
	}
	return cm, deployment
}

func testPluginTLSProfilesRendered(t testing.TB) {
	for _, tc := range []struct {
		name      string
		profile   map[string]any
		protocols string
		ciphers   string
	}{
		{"default", nil, "TLSv1.2 TLSv1.3", "ECDHE-RSA-AES128-GCM-SHA256"},
		{"modern", map[string]any{"type": "Modern"}, "TLSv1.3", "TLS_AES_128_GCM_SHA256"},
		{"custom", map[string]any{"type": "Custom", "custom": map[string]any{
			"minTLSVersion": "VersionTLS12",
			"ciphers":       []any{"ECDHE-RSA-AES256-GCM-SHA384", "TLS_AES_256_GCM_SHA384"},
			"groups":        []any{"secp256r1"},
		}}, "TLSv1.2 TLSv1.3", "ECDHE-RSA-AES256-GCM-SHA384"},
	} {
		By(tc.name)
		r, instance, cl, _ := setupPluginTLS(t, tc.profile)
		if err := r.applyManifests(instance, context.Background()); err != nil {
			t.Fatal(err)
		}
		cm, deployment := pluginTLSObjects(t, cl)
		conf := cm.Data["nginx.conf"]
		if !strings.Contains(conf, "ssl_protocols       "+tc.protocols+";") || !strings.Contains(conf, tc.ciphers) {
			t.Fatalf("cluster TLS policy missing from rendered nginx config:\n%s", conf)
		}
		hash := deployment.Spec.Template.Annotations["nmstate.io/tls-profile-hash"]
		if hash != fmt.Sprintf("%x", sha256.Sum256([]byte(conf))) {
			t.Fatal("plugin template is not bound to its rendered nginx configuration")
		}
		if !strings.Contains(conf, "ssl_certificate     $plugin_certificate_dir/tls.crt;") {
			t.Fatal("certificate hot loading was lost")
		}
		if cl.profileReads != 1 {
			t.Fatalf("expected one shared cluster profile read, got %d", cl.profileReads)
		}
	}
}

func testPluginTLSConfigRollout(t testing.TB) {
	r, instance, cl, dir := setupPluginTLS(t, map[string]any{"type": "Intermediate"})
	ctx := context.Background()
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	beforeConfig, beforeDeployment := pluginTLSObjects(t, cl)
	api := &unstructured.Unstructured{}
	api.SetAPIVersion("config.openshift.io/v1")
	api.SetKind("APIServer")
	if err := cl.Get(ctx, client.ObjectKey{Name: "cluster"}, api); err != nil {
		t.Fatal(err)
	}
	if err := unstructured.SetNestedField(api.Object, "Modern", "spec", "tlsSecurityProfile", "type"); err != nil {
		t.Fatal(err)
	}
	if err := cl.Update(ctx, api); err != nil {
		t.Fatal(err)
	}
	cl.failConfig = true
	if err := r.applyManifests(instance, ctx); err == nil || !strings.Contains(err.Error(), "injected plugin configuration failure") {
		t.Fatalf("expected configuration write failure, got %v", err)
	}
	_, during := pluginTLSObjects(t, cl)
	if !reflect.DeepEqual(beforeDeployment.Spec.Template, during.Spec.Template) {
		t.Fatal("new plugin template was published before its configuration")
	}
	cl.failConfig = false
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	afterConfig, afterDeployment := pluginTLSObjects(t, cl)
	if beforeConfig.Data["nginx.conf"] == afterConfig.Data["nginx.conf"] ||
		reflect.DeepEqual(beforeDeployment.Spec.Template, afterDeployment.Spec.Template) {
		t.Fatal("TLS profile change did not update both config and plugin template")
	}
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	_, unchanged := pluginTLSObjects(t, cl)
	if !reflect.DeepEqual(afterDeployment.Spec.Template, unchanged.Spec.Template) {
		t.Fatal("unchanged configuration triggers another rollout")
	}
	// An operator upgrade can change nginx configuration without changing the
	// cluster profile. That change must also activate on existing pods.
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const path = "kubernetes-nmstate/openshift/ui-plugin/configmap.yaml"
	data, err := root.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "keepalive_timeout  65;", "keepalive_timeout  60;", 1)
	if err := root.WriteFile(path, []byte(updated), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.applyManifests(instance, ctx); err != nil {
		t.Fatal(err)
	}
	_, upgraded := pluginTLSObjects(t, cl)
	if reflect.DeepEqual(unchanged.Spec.Template, upgraded.Spec.Template) {
		t.Fatal("nginx config change on upgrade did not trigger replacement")
	}
}

func testPluginTLSPartialProfileReportsWarning(t testing.TB) {
	r, instance, cl, _ := setupPluginTLS(t, map[string]any{
		"type": "Custom",
		"custom": map[string]any{
			"minTLSVersion": "VersionTLS12",
			"ciphers":       []any{"ECDHE-RSA-AES128-GCM-SHA256", "DHE-RSA-AES128-GCM-SHA256"},
			"groups":        []any{"X25519", "unknown-group"},
		},
	})
	if err := r.applyManifests(instance, context.Background()); err != nil {
		t.Fatal(err)
	}
	conf, _ := pluginTLSObjects(t, cl)
	if !strings.Contains(conf.Data["nginx.conf"], "ssl_ecdh_curve      X25519;") ||
		strings.Contains(conf.Data["nginx.conf"], "unknown-group") {
		t.Fatal("partial profile was not rendered with its supported groups")
	}
	if !strings.Contains(r.tlsProfileWarning, "console plugin") ||
		!strings.Contains(r.tlsProfileWarning, "DHE-RSA-AES128-GCM-SHA256") ||
		!strings.Contains(r.tlsProfileWarning, "unknown-group") {
		t.Fatalf("missing plugin policy warning: %s", r.tlsProfileWarning)
	}
}

func testPluginTLSInvalidProfileDoesNotApplyPlugin(t testing.TB) {
	r, instance, cl, _ := setupPluginTLS(t, map[string]any{
		"type": "Custom",
		"custom": map[string]any{
			"minTLSVersion": "VersionTLS13",
			"ciphers":       []any{"ECDHE-RSA-AES128-GCM-SHA256"},
		},
	})
	err := r.applyManifests(instance, context.Background())
	if err == nil || !strings.Contains(err.Error(), "console plugin TLS profile cannot be honored") {
		t.Fatalf("expected explicit rejection of incompatible TLS version/ciphers: %v", err)
	}
	configMaps := &corev1.ConfigMapList{}
	if err := cl.List(context.Background(), configMaps, client.InNamespace("nmstate")); err != nil {
		t.Fatal(err)
	}
	for _, cm := range configMaps.Items {
		if strings.HasPrefix(cm.Name, pluginNginxConfigName) {
			t.Fatalf("invalid profile was applied to plugin as ConfigMap %s", cm.Name)
		}
	}
}
