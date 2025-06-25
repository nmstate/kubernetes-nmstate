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
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nmstate/kubernetes-nmstate/pkg/apply"
)

func unstructuredFromYaml(obj string) *unstructured.Unstructured {
	buf := bytes.NewBufferString(obj)
	decoder := yaml.NewYAMLOrJSONDecoder(buf, 4096)

	u := unstructured.Unstructured{}
	err := decoder.Decode(&u)
	if err != nil {
		panic(fmt.Sprintf("failed to parse test yaml: %v", err))
	}

	return &u
}

var _ = Describe("ApplyObject", func() {
	Context("when server is empty", func() {
		var client k8sclient.Client
		BeforeEach(func() {
			objs := []runtime.Object{}
			client = fake.NewFakeClient(objs...)
		})

		Context("and new object is applied", func() {
			object := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1`)

			BeforeEach(func() {
				err := apply.ApplyObject(context.Background(), client, object)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create new object", func() {
				err := client.Get(context.Background(), types.NamespacedName{Name: "d1"}, &appsv1.Deployment{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("when server has object", func() {
		var client k8sclient.Client
		var originalDeployment *unstructured.Unstructured
		BeforeEach(func() {
			originalDeployment = unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  creationTimestamp: 2019-06-12T13:49:20Z
  generation: 1
  resourceVersion: "439"
  selfLink: /apis/extensions/v1beta1/namespaces/kube-system/deployments/d1
  uid: e0ecf168-8d18-11e9-b398-525500d15501
  annotations:
    foo: A`)

			objs := []runtime.Object{originalDeployment}
			client = fake.NewFakeClient(objs...)
		})

		Context("and is given the same object with same annotations", func() {
			found := &appsv1.Deployment{}
			object := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  annotations:
    foo: A`)

			It("should successfully merge", func() {
				By("Apllying object to server")
				err := apply.ApplyObject(context.Background(), client, object)
				Expect(err).ToNot(HaveOccurred())

				By("Finding the object in server")
				err = client.Get(context.Background(), types.NamespacedName{Name: "d1"}, found)
				Expect(err).ToNot(HaveOccurred())

				By("Having same metadata")
				Expect(found.GetCreationTimestamp()).To(Equal(originalDeployment.GetCreationTimestamp()))
				Expect(found.GetGeneration()).To(Equal(originalDeployment.GetGeneration()))
				Expect(found.GetSelfLink()).To(Equal(originalDeployment.GetSelfLink()))
				Expect(found.GetUID()).To(Equal(originalDeployment.GetUID()))
			})
		})

		Context("and is given the same object with different annotations", func() {
			found := &appsv1.Deployment{}
			object := unstructuredFromYaml(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  annotations:
    foo: B`)

			It("should have new annotations", func() {
				By("Apllying object to server")
				err := apply.ApplyObject(context.Background(), client, object)
				Expect(err).ToNot(HaveOccurred())

				By("Finding the object in server")
				err = client.Get(context.Background(), types.NamespacedName{Name: "d1"}, found)
				Expect(err).ToNot(HaveOccurred())

				By("Having the updated annotations")
				Expect(found.GetAnnotations()).To(Equal(object.GetAnnotations()))
			})
		})
	})
})
