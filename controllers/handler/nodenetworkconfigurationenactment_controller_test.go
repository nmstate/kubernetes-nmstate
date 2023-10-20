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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstateenactment "github.com/nmstate/kubernetes-nmstate/pkg/enactment"
)

var _ = Describe("Node Network Configuration Enactment controller reconcile", func() {
	var (
		cl         client.Client
		reconciler NodeNetworkConfigurationEnactmentReconciler
		policy     = nmstatev1.NodeNetworkConfigurationPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nmstate.io/v1",
				Kind:       "NodeNetworkConfigurationPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy1",
				UID:  "12345",
			},
		}
		enactment = nmstatev1beta1.NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   shared.EnactmentKey("node01", policy.Name).Name,
				Labels: map[string]string{shared.EnactmentPolicyLabel: policy.Name},
			},
		}
		expectRequeueAfterIsSetWithEnactmentRefresh = func(result ctrl.Result) {
			ExpectWithOffset(1, result.RequeueAfter).
				To(
					BeNumerically(
						"~",
						nmstateenactment.EnactmentRefresh,
						float64(nmstateenactment.EnactmentRefresh)*nmstateenactment.EnactmentRefreshMaxFactor,
					),
				)
		}
	)
	BeforeEach(func() {
		reconciler = NodeNetworkConfigurationEnactmentReconciler{}
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
		)
		s.AddKnownTypes(nmstatev1.GroupVersion,
			&nmstatev1.NodeNetworkConfigurationPolicy{},
		)

		objs := []runtime.Object{&policy, &enactment}

		// Create a fake client to mock API calls.
		cl = fake.
			NewClientBuilder().
			WithScheme(s).
			WithRuntimeObjects(objs...).
			WithStatusSubresource(&nmstatev1beta1.NodeNetworkConfigurationEnactment{}).
			Build()

		reconciler.Client = cl
		reconciler.Log = ctrl.Log.WithName("controllers").WithName("Enactment")
		reconciler.Scheme = s
	})
	Context("and policy exists", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = enactment.Name
		})
		It("should re-enqueue", func() {
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			expectRequeueAfterIsSetWithEnactmentRefresh(result)
		})
	})
	Context("and policy doesn't exist", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = enactment.Name

			By("Delete the policy")
			err := cl.Delete(context.TODO(), &policy)
			Expect(err).ToNot(HaveOccurred())

		})
		It("should remove the enactment", func() {
			_, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())

			obtainedEnactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: enactment.Name}, &obtainedEnactment)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
	Context("and enactment is not found", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = "not-present-enactment"
		})
		It("should returns empty result", func() {
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
