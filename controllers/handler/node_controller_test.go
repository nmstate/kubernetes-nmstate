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
	"fmt"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
)

var _ = Describe("Node controller reconcile", func() {
	var (
		cl                       client.Client
		reconciler               NodeReconciler
		observedState            string
		filteredOutObservedState shared.State
		existingNodeName         = "node01"
		node                     = corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: existingNodeName,
				UID:  "12345",
			},
		}
		nodenetworkstate = nmstatev1beta1.NodeNetworkState{
			ObjectMeta: metav1.ObjectMeta{
				Name: existingNodeName,
			},
		}
		expectRequeueAfterIsSetWithNetworkStateRefresh = func(result ctrl.Result) {
			ExpectWithOffset(1, result.RequeueAfter).
				To(
					BeNumerically(
						"~",
						nmstatenode.NetworkStateRefresh,
						float64(nmstatenode.NetworkStateRefresh)*nmstatenode.NetworkStateRefreshMaxFactor,
					),
				)
		}
	)
	BeforeEach(func() {
		reconciler = NodeReconciler{}
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkState{},
		)

		objs := []runtime.Object{&node, &nodenetworkstate}

		// Create a fake client to mock API calls.
		cl = fake.NewClientBuilder().WithScheme(s).WithStatusSubresource(&nodenetworkstate).WithRuntimeObjects(objs...).Build()

		reconciler.Client = cl
		reconciler.Log = ctrl.Log.WithName("controllers").WithName("Node")
		reconciler.Scheme = s
		reconciler.nmstateUpdater = nmstate.CreateOrUpdateNodeNetworkState
		reconciler.nmstatectlShow = nmstatectl.Show
		reconciler.lastState = shared.NewState("lastState")
		observedState = `
---
interfaces:
  - name: eth1
    type: ethernet
    state: up
routes:
  running: []
  config: []
`

		var err error
		filteredOutObservedState, err = state.FilterOut(shared.NewState(observedState))
		Expect(err).ToNot(HaveOccurred())

		reconciler.nmstatectlShow = func() (string, error) {
			return observedState, nil
		}
	})
	Context("and nmstatectl show is failing", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			reconciler.nmstatectlShow = func() (string, error) {
				return "", fmt.Errorf("forced failure at unit test")
			}
		})
		It("should return the error from nmstatectl", func() {
			_, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).To(MatchError("forced failure at unit test"))
		})
	})
	Context("and network state didn't change", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			By("Set last state")
			reconciler.lastState = filteredOutObservedState

			reconciler.nmstateUpdater = func(client.Client, *corev1.Node,
				shared.State, *nmstatev1beta1.NodeNetworkState, *nmstate.DependencyVersions) error {
				return fmt.Errorf("we are not suppose to catch this error")
			}

			request.Name = existingNodeName
		})
		It("should not call nmstateUpdater and return a Result with RequeueAfter set", func() {
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			expectRequeueAfterIsSetWithNetworkStateRefresh(result)
		})
	})
	Context("when node is not found", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = "not-present-node"
		})
		It("should returns empty result", func() {
			result, err := reconciler.Reconcile(context.Background(), request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
	Context("when a node is found", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			request.Name = existingNodeName
		})
		Context(", nodenetworkstate is there too with last state and observed state is different", func() {
			var (
				expectedStateRaw = `---
interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: eth2
    type: ethernet
    state: up
routes:
  running: []
  config: []
`
			)

			BeforeEach(func() {
				By("Set last state")
				reconciler.lastState = filteredOutObservedState

				By("Mock nmstate show so we return different value from last state")
				reconciler.nmstatectlShow = func() (string, error) {
					return expectedStateRaw, nil
				}

			})
			It("should call nmstateUpdater and return a Result with RequeueAfter set (trigger re-reconciliation)", func() {
				result, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectRequeueAfterIsSetWithNetworkStateRefresh(result)
				obtainedNNS := nmstatev1beta1.NodeNetworkState{}
				err = cl.Get(context.TODO(), types.NamespacedName{Name: existingNodeName}, &obtainedNNS)
				Expect(err).ToNot(HaveOccurred())
				filteredOutExpectedState, err := state.FilterOut(shared.NewState(expectedStateRaw))
				Expect(err).ToNot(HaveOccurred())
				Expect(obtainedNNS.Status.CurrentState.String()).To(Equal(filteredOutExpectedState.String()))
			})
		})
		Context("and nodenetworkstate is not there", func() {
			BeforeEach(func() {
				By("Delete the nodenetworkstate")
				err := cl.Delete(context.TODO(), &nodenetworkstate)
				Expect(err).ToNot(HaveOccurred())

				By("Set last state")
				reconciler.lastState = filteredOutObservedState
			})
			It(
				"should create a new nodenetworkstate with node as owner reference, making sure "+
					"the nodenetworkstate will be removed when the node is deleted",
				func() {
					_, err := reconciler.Reconcile(context.Background(), request)
					Expect(err).ToNot(HaveOccurred())

					obtainedNNS := nmstatev1beta1.NodeNetworkState{}
					nnsKey := types.NamespacedName{Name: existingNodeName}
					err = cl.Get(context.TODO(), types.NamespacedName{Name: existingNodeName}, &obtainedNNS)
					Expect(err).ToNot(HaveOccurred())
					Expect(obtainedNNS.Name).To(Equal(nnsKey.Name))
					Expect(obtainedNNS.ObjectMeta.OwnerReferences).To(HaveLen(1))
					Expect(obtainedNNS.ObjectMeta.OwnerReferences[0]).To(Equal(
						metav1.OwnerReference{Name: existingNodeName, Kind: "Node", APIVersion: "v1", UID: node.UID},
					))
				},
			)
			It("should return a Result with RequeueAfter set (trigger re-reconciliation)", func() {
				result, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectRequeueAfterIsSetWithNetworkStateRefresh(result)
			})
		})
	})
})
