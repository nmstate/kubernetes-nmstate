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
		reconciler.nmstatectlShow = nmstatectl.ShowWithTimeout
		reconciler.nmstatectlShowKernel = func(context.Context) error { return nil }
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

		reconciler.nmstatectlShow = func(context.Context) (string, error) {
			return observedState, nil
		}
	})
	Context("and nmstatectl show is failing", func() {
		var (
			request reconcile.Request
		)
		getNNS := func() nmstatev1beta1.NodeNetworkState {
			obtainedNNS := nmstatev1beta1.NodeNetworkState{}
			ExpectWithOffset(1, cl.Get(context.TODO(), types.NamespacedName{Name: existingNodeName}, &obtainedNNS)).To(Succeed())
			return obtainedNNS
		}
		expectConditions := func(nns nmstatev1beta1.NodeNetworkState, available, failing corev1.ConditionStatus,
			reason shared.ConditionReason) {
			availableCondition := nns.Status.Conditions.Find(shared.NodeNetworkStateConditionAvailable)
			ExpectWithOffset(1, availableCondition).ToNot(BeNil())
			ExpectWithOffset(1, availableCondition.Status).To(Equal(available))
			ExpectWithOffset(1, availableCondition.Reason).To(Equal(reason))
			failingCondition := nns.Status.Conditions.Find(shared.NodeNetworkStateConditionFailing)
			ExpectWithOffset(1, failingCondition).ToNot(BeNil())
			ExpectWithOffset(1, failingCondition.Status).To(Equal(failing))
			ExpectWithOffset(1, failingCondition.Reason).To(Equal(reason))
		}
		BeforeEach(func() {
			request.Name = existingNodeName
			reconciler.nmstatectlShow = func(context.Context) (string, error) {
				return "", fmt.Errorf("forced failure at unit test")
			}
		})
		Context("and kernel-only query works", func() {
			It("should report NetworkManagerUnresponsive and requeue at the refresh interval", func() {
				result, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectRequeueAfterIsSetWithNetworkStateRefresh(result)
				nns := getNNS()
				expectConditions(nns, corev1.ConditionFalse, corev1.ConditionTrue,
					shared.NodeNetworkStateConditionNetworkManagerUnresponsive)
				Expect(nns.Status.Conditions.Find(shared.NodeNetworkStateConditionFailing).Message).
					To(ContainSubstring("forced failure at unit test"))
			})
		})
		Context("and kernel-only query fails too", func() {
			BeforeEach(func() {
				reconciler.nmstatectlShowKernel = func(context.Context) error {
					return fmt.Errorf("forced kernel failure at unit test")
				}
			})
			It("should report QueryFailed with both errors", func() {
				_, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				nns := getNNS()
				expectConditions(nns, corev1.ConditionFalse, corev1.ConditionTrue,
					shared.NodeNetworkStateConditionQueryFailed)
				message := nns.Status.Conditions.Find(shared.NodeNetworkStateConditionFailing).Message
				Expect(message).To(ContainSubstring("forced failure at unit test"))
				Expect(message).To(ContainSubstring("forced kernel failure at unit test"))
			})
		})
		Context("and there is a previously stored state", func() {
			BeforeEach(func() {
				nns := getNNS()
				nns.Status.CurrentState = filteredOutObservedState
				Expect(cl.Status().Update(context.TODO(), &nns)).To(Succeed())
			})
			It("should keep the last good state", func() {
				_, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(getNNS().Status.CurrentState.String()).To(Equal(filteredOutObservedState.String()))
			})
			It("should report success again once the query recovers", func() {
				_, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectConditions(getNNS(), corev1.ConditionFalse, corev1.ConditionTrue,
					shared.NodeNetworkStateConditionNetworkManagerUnresponsive)

				By("Recover nmstatectl show with the same state as stored")
				reconciler.lastState = filteredOutObservedState
				reconciler.nmstatectlShow = func(context.Context) (string, error) {
					return observedState, nil
				}
				_, err = reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectConditions(getNNS(), corev1.ConditionTrue, corev1.ConditionFalse,
					shared.NodeNetworkStateConditionQuerySucceeded)
			})
		})
		Context("and nodenetworkstate is not there", func() {
			BeforeEach(func() {
				Expect(cl.Delete(context.TODO(), &nodenetworkstate)).To(Succeed())
			})
			It("should create it with the failure conditions", func() {
				_, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				expectConditions(getNNS(), corev1.ConditionFalse, corev1.ConditionTrue,
					shared.NodeNetworkStateConditionNetworkManagerUnresponsive)
			})
		})
		Context("and node is not found", func() {
			BeforeEach(func() {
				request.Name = "not-present-node"
			})
			It("should return empty result", func() {
				result, err := reconciler.Reconcile(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
			})
		})
	})
	Context("and network state didn't change", func() {
		var (
			request reconcile.Request
		)
		BeforeEach(func() {
			By("Set last state")
			reconciler.lastState = filteredOutObservedState

			reconciler.nmstateUpdater = func(context.Context, client.Client, *corev1.Node,
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
				reconciler.nmstatectlShow = func(context.Context) (string, error) {
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
				available := obtainedNNS.Status.Conditions.Find(shared.NodeNetworkStateConditionAvailable)
				Expect(available).ToNot(BeNil())
				Expect(available.Status).To(Equal(corev1.ConditionTrue))
				Expect(available.Reason).To(Equal(shared.NodeNetworkStateConditionQuerySucceeded))
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
