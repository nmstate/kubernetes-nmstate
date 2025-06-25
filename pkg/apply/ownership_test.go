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

package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("Ownership functions", func() {
	Context("when testing nmstateOwns function", func() {
		It("should return true when object is owned by NMState", func() {
			obj := &uns.Unstructured{}
			obj.SetOwnerReferences([]metav1.OwnerReference{
				{
					Kind: "NMState",
					Name: "test-nmstate",
					UID:  "12345",
				},
			})

			Expect(nmstateOwns(obj)).To(BeTrue())
		})

		It("should return false when object is not owned by NMState", func() {
			obj := &uns.Unstructured{}
			obj.SetOwnerReferences([]metav1.OwnerReference{
				{
					Kind: "SomeOtherKind",
					Name: "test-other",
					UID:  "67890",
				},
			})

			Expect(nmstateOwns(obj)).To(BeFalse())
		})

		It("should return false when object has no owner references", func() {
			obj := &uns.Unstructured{}

			Expect(nmstateOwns(obj)).To(BeFalse())
		})

		It("should return true when object has multiple owners including NMState", func() {
			obj := &uns.Unstructured{}
			obj.SetOwnerReferences([]metav1.OwnerReference{
				{
					Kind: "SomeOtherKind",
					Name: "test-other",
					UID:  "67890",
				},
				{
					Kind: "NMState",
					Name: "test-nmstate",
					UID:  "12345",
				},
			})

			Expect(nmstateOwns(obj)).To(BeTrue())
		})
	})

	Context("when testing isTLSSecret function", func() {
		It("should return true for TLS secret", func() {
			obj := &uns.Unstructured{}
			obj.SetKind("Secret")
			obj.Object = map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"type":       "kubernetes.io/tls",
			}

			Expect(isTLSSecret(obj)).To(BeTrue())
		})

		It("should return false for non-TLS secret", func() {
			obj := &uns.Unstructured{}
			obj.SetKind("Secret")
			obj.Object = map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"type":       "Opaque",
			}

			Expect(isTLSSecret(obj)).To(BeFalse())
		})

		It("should return false for non-Secret object", func() {
			obj := &uns.Unstructured{}
			obj.SetKind("ConfigMap")

			Expect(isTLSSecret(obj)).To(BeFalse())
		})

		It("should return false for Secret without type", func() {
			obj := &uns.Unstructured{}
			obj.SetKind("Secret")
			obj.Object = map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
			}

			Expect(isTLSSecret(obj)).To(BeFalse())
		})
	})
})
