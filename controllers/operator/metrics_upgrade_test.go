//go:build integration

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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

// A real API server is required: replacing SSA with Update drops even containers
// that are still owned by a legacy writer, masking the upgrade failure.
func TestMetricsUpgrade(t *testing.T) {
	g := NewWithT(t)
	controlPlane := &envtest.Environment{}
	cfg, err := controlPlane.Start()
	g.Expect(err).NotTo(HaveOccurred())
	t.Cleanup(func() { g.Expect(controlPlane.Stop()).To(Succeed()) })
	g.Expect(nmstatev1.AddToScheme(scheme.Scheme)).To(Succeed())
	cl, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	g.Expect(err).NotTo(HaveOccurred())
	r := NMStateReconciler{Client: cl, APIClient: cl, Scheme: scheme.Scheme}
	instance := &nmstatev1.NMState{ObjectMeta: metav1.ObjectMeta{Name: "nmstate", UID: "test-nmstate"}}

	previousDir := names.ManifestDir
	names.ManifestDir = t.TempDir()
	t.Cleanup(func() { names.ManifestDir = previousDir })
	dir := filepath.Join(names.ManifestDir, "kubernetes-nmstate", "handler")
	g.Expect(os.MkdirAll(dir, 0755)).To(Succeed())
	// Exercise handler reconciliation with just the desired metrics Deployment.
	g.Expect(os.WriteFile(filepath.Join(dir, "metrics.yaml"), []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .HandlerPrefix }}-nmstate-metrics
  namespace: {{ .HandlerNamespace }}
  labels: {component: kubernetes-nmstate-metrics}
spec:
  selector: {matchLabels: {app: metrics}}
  template:
    metadata: {labels: {app: metrics}}
    spec:
      containers:
      - name: nmstate-metrics
        image: manager:new
        env: [{name: METRICS_BIND_ADDRESS, value: ":8443"}]
`), 0600)).To(Succeed())
	t.Setenv("HANDLER_NAMESPACE", "default")

	for _, prefix := range []string{"same-image", "changed-image"} {
		t.Run(prefix, func(t *testing.T) {
			g := NewWithT(t)
			ctx := t.Context()
			t.Setenv("HANDLER_PREFIX", prefix)
			labels := map[string]string{"app": "metrics"}
			key := client.ObjectKey{Namespace: "default", Name: prefix + "-nmstate-metrics"}
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace,
					OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(instance, nmstatev1.GroupVersion.WithKind("NMState"))}},
				Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: labels},
					Spec: corev1.PodSpec{Containers: []corev1.Container{
						{Name: "nmstate-metrics", Image: "manager:old"},
						{Name: "kube-rbac-proxy", Image: "proxy:old"},
						{Name: "other", Image: "other:keep"},
					}},
				}},
			}
			g.Expect(cl.Create(ctx, dep, client.FieldOwner("legacy-writer"))).To(Succeed())
			image := "proxy:old"
			if prefix == "changed-image" {
				image = "proxy:updated"
			}
			patch := []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"` + key.Name +
				`","namespace":"default"},"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","image":"` + image + `"}]}}}}`)
			g.Expect(cl.Patch(ctx, dep, client.RawPatch(types.ApplyPatchType, patch), nmstateOperatorFieldOwner, client.ForceOwnership)).To(Succeed())

			g.Expect(r.applyHandler(ctx, instance, nil)).To(Succeed())
			g.Expect(cl.Get(ctx, key, dep)).To(Succeed())
			g.Expect(dep.Spec.Template.Spec.Containers).To(ConsistOf(
				HaveField("Image", "manager:new"), HaveField("Image", "other:keep")))
			generation := dep.Generation
			g.Expect(r.applyHandler(ctx, instance, nil)).To(Succeed())
			g.Expect(cl.Get(ctx, key, dep)).To(Succeed())
			g.Expect(dep.Generation).To(Equal(generation))
		})
	}
}
