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

package apply_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/nmstate/kubernetes-nmstate/pkg/apply"
)

var _ = Describe("MergeObjectForUpdate", func() {
	// Namespaces use the "generic" logic; deployments and services
	// have custom logic
	Context("when given a generic object (Namespace)", func() {
		cur := unstructuredFromYaml(`
apiVersion: v1
kind: Namespace
metadata:
  name: ns1
  labels:
    a: cur
    b: cur
  annotations:
    a: cur
    b: cur`)

		upd := unstructuredFromYaml(`
apiVersion: v1
kind: Namespace
metadata:
  name: ns1
  labels:
    a: upd
    c: upd
  annotations:
    a: upd
    c: upd`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should merge annotations", func() {
			Expect(upd.GetLabels()).To(Equal(map[string]string{
				"a": "upd",
				"b": "cur",
				"c": "upd",
			}))
		})

		It("should overwrite everything else", func() {
			Expect(upd.GetAnnotations()).To(Equal(map[string]string{
				"a": "upd",
				"b": "cur",
				"c": "upd",
			}))
		})
	})

	Context("when given a Deployment", func() {
		cur := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  labels:
    a: cur
    b: cur
  annotations:
    deployment.kubernetes.io/revision: cur
    a: cur
    b: cur`)

		upd := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  labels:
    a: upd
    c: upd
  annotations:
    deployment.kubernetes.io/revision: upd
    a: upd
    c: upd`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should merge annotations", func() {
			Expect(upd.GetAnnotations()).To(Equal(map[string]string{
				"a": "upd",
				"b": "cur",
				"c": "upd",

				"deployment.kubernetes.io/revision": "cur",
			}))
		})

		It("should not merge labels", func() {
			Expect(upd.GetLabels()).To(Equal(map[string]string{
				"a": "upd",
				"b": "cur",
				"c": "upd",
			}))
		})
	})

	Context("when given a Service", func() {
		cur := unstructuredFromYaml(`
apiVersion: v1
kind: Service
metadata:
  name: d1
spec:
  clusterIP: cur`)

		upd := unstructuredFromYaml(`
apiVersion: v1
kind: Service
metadata:
  name: d1
spec:
  clusterIP: upd`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should keep the original clusterIP", func() {
			ip, _, err := unstructured.NestedString(upd.Object, "spec", "clusterIP")
			Expect(err).NotTo(HaveOccurred())
			Expect(ip).To(Equal("cur"))
		})
	})

	Context("when given a ServiceAccount", func() {
		cur := unstructuredFromYaml(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: d1
  annotations:
    a: cur
secrets:
- foo`)

		upd := unstructuredFromYaml(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: d1
  annotations:
    b: upd`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should keep original secrets after merging", func() {
			s, ok, err := unstructured.NestedSlice(upd.Object, "secrets")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(s).To(ConsistOf("foo"))
		})
	})

	Context("when merging an empty Deployment into an empty Deployment", func() {
		cur := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1`)

		upd := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should stay empty", func() {
			Expect(upd.GetLabels()).To(BeEmpty())
		})
	})

	Context("when merging a non-empty Deployment into an empty Deployment", func() {
		cur := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1`)

		upd := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  labels:
    a: upd
    c: upd
  annotations:
    a: upd
    c: upd`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should use values from the updating Deployment", func() {
			Expect(upd.GetLabels()).To(Equal(map[string]string{
				"a": "upd",
				"c": "upd",
			}))

			Expect(upd.GetAnnotations()).To(Equal(map[string]string{
				"a": "upd",
				"c": "upd",
			}))
		})
	})

	Context("when merging an empty Deployment into a non-empty Deployment", func() {
		cur := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  labels:
    a: cur
    b: cur
  annotations:
    a: cur
    b: cur`)

		upd := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1`)

		It("should successfully merge", func() {
			// this mutates updated
			err := apply.MergeObjectForUpdate(cur, upd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("keep the original values and not overwrite them with pure void and emptiness", func() {
			Expect(upd.GetLabels()).To(Equal(map[string]string{
				"a": "cur",
				"b": "cur",
			}))

			Expect(upd.GetAnnotations()).To(Equal(map[string]string{
				"a": "cur",
				"b": "cur",
			}))
		})
	})
	Context("when merging webhookconfiguration", func() {
		type webhookConfig struct {
			app string
			wh1 string
			wh2 string
			wh3 string
		}
		type webhookConfigCase struct {
			current  webhookConfig
			updated  webhookConfig
			expected webhookConfig
		}
		var (
			template = `
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: nmstate
  labels:
    app: %s
  annotations: {}
webhooks:
  - name: nodenetworkconfigurationpolicies-mutate.nmstate.io
    clientConfig:
      %s
      service:
        name: nmstate-webhook
        namespace: nmstate
        path: "/nodenetworkconfigurationpolicies-mutate"
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["v1alpha1"]
        resources: ["nodenetworkconfigurationpolicies"]
  - name: nodenetworkconfigurationpolicies-status-mutate.nmstate.io
    clientConfig:
      %s
      service:
        name: nmstate-webhook
        namespace: nmstate
        path: "/nodenetworkconfigurationpolicies-status-mutate"
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["v1alpha1"]
        resources: ["nodenetworkconfigurationpolicies/status"]
  - name: nodenetworkconfigurationpolicies-timestamp-mutate.nmstate.io
    clientConfig:
      %s
      service:
        name: nmstate-webhook
        namespace: nmstate
        path: "/nodenetworkconfigurationpolicies-timestamp-mutate"
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["v1alpha1"]
        resources: ["nodenetworkconfigurationpolicies", "nodenetworkconfigurationpolicies/status"]
`
			generateUnstructured = func(config webhookConfig) *unstructured.Unstructured {
				return unstructuredFromYaml(fmt.Sprintf(template, config.app, config.wh1, config.wh2, config.wh3))
			}
		)
		DescribeTable("and have modified caBundle and app label", func(c webhookConfigCase) {
			current := generateUnstructured(c.current)
			updated := generateUnstructured(c.updated)
			expected := generateUnstructured(c.expected)

			err := apply.MergeObjectForUpdate(current, updated)
			Expect(err).ToNot(HaveOccurred(), "should successfully execut merge function")

			Expect(*updated).To(Equal(*expected), "the object should be updated as expected, with original caBundles left intact")
		},
			Entry("with caBundle non-empty at current config but not preset "+
				"at updated one, should preserve caBundle and update app label", webhookConfigCase{
				current: webhookConfig{
					app: "kubemacpool-1",
					wh1: "caBundle: cawh1",
					wh2: "caBundle: cawh2",
					wh3: "caBundle: cawh3",
				},
				updated: webhookConfig{
					app: "kubemacpool-2",
					wh1: "",
					wh2: "",
					wh3: "",
				},
				expected: webhookConfig{
					app: "kubemacpool-2",
					wh1: "caBundle: cawh1",
					wh2: "caBundle: cawh2",
					wh3: "caBundle: cawh3",
				},
			}),
			Entry("with caBundle not present at current config and non-empty at updated one, should use updated caBundle", webhookConfigCase{
				current: webhookConfig{
					app: "kubemacpool-1",
					wh1: "",
					wh2: "",
					wh3: "",
				},
				updated: webhookConfig{
					app: "kubemacpool-2",
					wh1: "caBundle: cawh1",
					wh2: "caBundle: cawh2",
					wh3: "caBundle: cawh3",
				},
				expected: webhookConfig{
					app: "kubemacpool-2",
					wh1: "caBundle: cawh1",
					wh2: "caBundle: cawh2",
					wh3: "caBundle: cawh3",
				},
			}),
			Entry("with different caBundle at updated, should use the new one", webhookConfigCase{
				current: webhookConfig{
					app: "kubemacpool-1",
					wh1: "caBundle: cawh1",
					wh2: "caBundle: cawh2",
					wh3: "caBundle: cawh3",
				},
				updated: webhookConfig{
					app: "kubemacpool-2",
					wh1: "caBundle: cawh1u",
					wh2: "caBundle: cawh2u",
					wh3: "caBundle: cawh3u",
				},
				expected: webhookConfig{
					app: "kubemacpool-2",
					wh1: "caBundle: cawh1u",
					wh2: "caBundle: cawh2u",
					wh3: "caBundle: cawh3u",
				},
			}),
		)
	})
})

var _ = Describe("IsObjectSupported", func() {
	Context("when given a ServiceAccount with a secret", func() {
		sa := unstructuredFromYaml(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: d1
  annotations:
    a: cur
secrets:
- foo`)

		It("should return an error", func() {
			err := apply.IsObjectSupported(sa)
			Expect(err).To(MatchError(ContainSubstring("cannot create ServiceAccount with secrets")))
		})
	})
})

var _ = Describe("MergeMetadataForUpdate", func() {
	Context("when given current unstructured and empty updated", func() {
		current := unstructuredFromYaml(`
apiVersion: v1
kind: Deployment
metadata:
  name: foo
  creationTimestamp: 2019-06-12T13:49:20Z
  generation: 1
  resourceVersion: "439"
  selfLink: /apis/extensions/v1beta1/namespaces/kube-system/deployments/foo
  uid: e0ecf168-8d18-11e9-b398-525500d15501
`)
		updated := unstructuredFromYaml(`
apiVersion: v1
kind: Deployment
metadata:
  name: foo`)

		It("should merge metadate from current to updated", func() {
			err := apply.MergeMetadataForUpdate(current, updated)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.GetCreationTimestamp()).To(Equal(current.GetCreationTimestamp()))
			Expect(updated.GetGeneration()).To(Equal(current.GetGeneration()))
			Expect(updated.GetResourceVersion()).To(Equal(current.GetResourceVersion()))
			Expect(updated.GetSelfLink()).To(Equal(current.GetSelfLink()))
			Expect(updated.GetUID()).To(Equal(current.GetUID()))
		})
	})
})
