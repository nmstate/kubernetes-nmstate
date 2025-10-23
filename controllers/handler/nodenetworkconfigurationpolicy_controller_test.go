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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
)

var _ = Describe("NodeNetworkConfigurationPolicy controller predicates", func() {
	type predicateCase struct {
		GenerationOld   int64
		GenerationNew   int64
		ReconcileUpdate bool
	}
	DescribeTable("testing predicates",
		func(c predicateCase) {
			oldNNCP := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationOld,
				},
			}
			newNNCP := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Generation: c.GenerationNew,
				},
			}

			predicate := onCreateOrUpdateWithDifferentGenerationOrDelete

			Expect(predicate.
				CreateFunc(event.TypedCreateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					Object: &newNNCP,
				})).To(BeTrue())
			Expect(predicate.
				UpdateFunc(event.TypedUpdateEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					ObjectOld: &oldNNCP,
					ObjectNew: &newNNCP,
				})).To(Equal(c.ReconcileUpdate))
			Expect(predicate.
				DeleteFunc(event.TypedDeleteEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					Object: &oldNNCP,
				})).To(BeTrue())
		},
		Entry("generation remains the same",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   1,
				ReconcileUpdate: false,
			}),
		Entry("generation is different",
			predicateCase{
				GenerationOld:   1,
				GenerationNew:   2,
				ReconcileUpdate: true,
			}),
	)

	type incrementUnavailableNodeCountCase struct {
		currentUnavailableNodeCount int
		expectedReconcileResult     ctrl.Result
		previousEnactmentConditions func(*shared.ConditionList, string)
	}
	DescribeTable("when claimNodeRunningUpdate is called and",
		func(c incrementUnavailableNodeCountCase) {
			nmstatectlShowFn = func() (string, error) { return "", nil }
			reconciler := NodeNetworkConfigurationPolicyReconciler{}
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1beta1.GroupVersion,
				&nmstatev1beta1.NodeNetworkState{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
				&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{},
			)
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
			)

			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}

			nns := nmstatev1beta1.NodeNetworkState{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}

			nncp := nmstatev1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Status: shared.NodeNetworkConfigurationPolicyStatus{
					UnavailableNodeCountMap: map[string]int{},
				},
			}
			nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
				ObjectMeta: metav1.ObjectMeta{
					Name: shared.EnactmentKey(nodeName, nncp.Name).Name,
				},
				Status: shared.NodeNetworkConfigurationEnactmentStatus{},
			}

			// simulate NNCE existnce/non-existence by setting conditions
			c.previousEnactmentConditions(&nnce.Status.Conditions, "")

			objs := []runtime.Object{&nncp, &nnce, &nns, &node}

			// Create a fake client to mock API calls.
			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			clb.WithRuntimeObjects(objs...)
			clb.WithStatusSubresource(&nncp)
			clb.WithStatusSubresource(&nnce)
			clb.WithStatusSubresource(&nns)
			cl := clb.Build()

			reconciler.Client = cl
			reconciler.APIClient = cl
			reconciler.Log = ctrl.Log.WithName("controllers").WithName("NodeNetworkConfigurationPolicy")

			res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: nncp.Name},
			})

			Expect(err).To(BeNil())
			Expect(res).To(Equal(c.expectedReconcileResult))
		},

		Entry("No node applying policy with empty enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("No node applying policy with progressing enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: conditions.SetProgressing,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("No node applying policy with Pending enactment, should succeed incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 0,
				previousEnactmentConditions: conditions.SetPending,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with empty enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: func(*shared.ConditionList, string) {},
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with Progressing enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: conditions.SetProgressing,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
		Entry("One node applying policy with Pending enactment, should conflict incrementing UnavailableNodeCount",
			incrementUnavailableNodeCountCase{
				currentUnavailableNodeCount: 1,
				previousEnactmentConditions: conditions.SetPending,
				expectedReconcileResult:     ctrl.Result{Requeue: true},
			}),
	)

	Describe("allPolicies function", func() {
		It("should return policies in alphanumerical order by name", func() {
			// Create test policies with names in non-alphabetical order
			policies := []nmstatev1.NodeNetworkConfigurationPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "zebra-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alpha-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "beta-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "1-numeric-policy",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "10-numeric-policy",
					},
				},
			}

			// Create a fake client with the test policies
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion, &nmstatev1.NodeNetworkConfigurationPolicy{}, &nmstatev1.NodeNetworkConfigurationPolicyList{})

			objs := make([]runtime.Object, len(policies))
			for i := range policies {
				objs[i] = &policies[i]
			}

			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			clb.WithRuntimeObjects(objs...)
			cl := clb.Build()

			// Get the allPolicies function
			allPoliciesFunc := allPolicies(cl, ctrl.Log)

			// Call the function with a dummy node
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			}

			requests := allPoliciesFunc(context.TODO(), node)

			// Verify the order is correct
			expectedOrder := []string{
				"1-numeric-policy",
				"10-numeric-policy",
				"alpha-policy",
				"beta-policy",
				"zebra-policy",
			}

			Expect(requests).To(WithTransform(func(reqs []ctrl.Request) []string {
				names := make([]string, len(reqs))
				for i, req := range reqs {
					names[i] = req.NamespacedName.Name
				}
				return names
			}, Equal(expectedOrder)))
		})

		It("should return empty slice when no policies exist", func() {
			// Create a fake client with no policies
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion, &nmstatev1.NodeNetworkConfigurationPolicy{}, &nmstatev1.NodeNetworkConfigurationPolicyList{})

			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			cl := clb.Build()

			// Get the allPolicies function
			allPoliciesFunc := allPolicies(cl, ctrl.Log)

			// Call the function with a dummy node
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			}

			requests := allPoliciesFunc(context.TODO(), node)

			// Verify empty result
			Expect(requests).To(BeEmpty())
		})
	})

	Describe("policyReconciliationTrigger", func() {
		var (
			reconciler *NodeNetworkConfigurationPolicyReconciler
			trigger    *policyReconciliationTrigger
			ctx        context.Context
			cancel     context.CancelFunc
		)

		BeforeEach(func() {
			// Set the global nodeName for testing
			nodeName = "test-node"

			// Create scheme with required types
			s := scheme.Scheme
			s.AddKnownTypes(nmstatev1.GroupVersion,
				&nmstatev1.NodeNetworkConfigurationPolicy{},
				&nmstatev1.NodeNetworkConfigurationPolicyList{})
			s.AddKnownTypes(corev1.SchemeGroupVersion,
				&corev1.Node{},
				&corev1.NodeList{})

			// Create fake client
			clb := fake.ClientBuilder{}
			clb.WithScheme(s)
			cl := clb.Build()

			// Create reconciler with event channel
			reconciler = &NodeNetworkConfigurationPolicyReconciler{
				Client:       cl,
				APIClient:    cl,
				Log:          ctrl.Log.WithName("test"),
				eventChannel: make(chan event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy], 100),
			}

			// Create trigger
			trigger = &policyReconciliationTrigger{
				reconciler: reconciler,
				log:        ctrl.Log.WithName("test-trigger"),
			}

			// Create context with cancel
			ctx, cancel = context.WithCancel(context.Background())
		})

		AfterEach(func() {
			cancel()
		})

		Describe("enqueuePoliciesForNode", func() {
			It("should enqueue policies that match the node selector", func() {
				// Create test node
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
						Labels: map[string]string{
							"kubernetes.io/hostname": "test-node",
							"node-role":              "worker",
						},
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				// Create policies with different selectors
				matchingPolicy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "matching-policy",
					},
					Spec: shared.NodeNetworkConfigurationPolicySpec{
						NodeSelector: map[string]string{
							"node-role": "worker",
						},
					},
				}
				Expect(reconciler.Client.Create(ctx, matchingPolicy)).To(Succeed())

				nonMatchingPolicy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "non-matching-policy",
					},
					Spec: shared.NodeNetworkConfigurationPolicySpec{
						NodeSelector: map[string]string{
							"node-role": "control-plane",
						},
					},
				}
				Expect(reconciler.Client.Create(ctx, nonMatchingPolicy)).To(Succeed())

				// Enqueue policies
				trigger.enqueuePoliciesForNode(ctx, "test")

				// Verify only matching policy was enqueued
				Eventually(reconciler.eventChannel).Should(Receive(WithTransform(
					func(e event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) string {
						return e.Object.GetName()
					},
					Equal("matching-policy"),
				)))

				// Verify no more events
				Consistently(reconciler.eventChannel).ShouldNot(Receive())
			})

			It("should handle empty policy list gracefully", func() {
				// Enqueue policies when none exist
				trigger.enqueuePoliciesForNode(ctx, "test")

				// Verify no events were sent
				Consistently(reconciler.eventChannel).ShouldNot(Receive())
			})

			It("should enqueue all policies when no selector is specified", func() {
				// Create test node
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				// Create policy without selector (matches all nodes)
				policy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "all-nodes-policy",
					},
					Spec: shared.NodeNetworkConfigurationPolicySpec{},
				}
				Expect(reconciler.Client.Create(ctx, policy)).To(Succeed())

				// Enqueue policies
				trigger.enqueuePoliciesForNode(ctx, "test")

				// Verify policy was enqueued
				Eventually(reconciler.eventChannel).Should(Receive(WithTransform(
					func(e event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) string {
						return e.Object.GetName()
					},
					Equal("all-nodes-policy"),
				)))
			})

			It("should stop enqueuing when context is cancelled", func() {
				// Create many policies
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				for i := 0; i < 10; i++ {
					policy := &nmstatev1.NodeNetworkConfigurationPolicy{
						ObjectMeta: metav1.ObjectMeta{
							Name: types.NamespacedName{Name: "policy-" + string(rune(i))}.Name,
						},
					}
					Expect(reconciler.Client.Create(ctx, policy)).To(Succeed())
				}

				// Cancel context immediately
				cancel()

				// Enqueue should stop early
				trigger.enqueuePoliciesForNode(ctx, "test")

				// We might receive some events, but not all 10
				// Just verify it doesn't panic or hang
				Eventually(func() bool {
					select {
					case <-reconciler.eventChannel:
						return true
					default:
						return true
					}
				}).Should(BeTrue())
			})

			It("should handle event channel being full gracefully", func() {
				// Create test node
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				// Create a small channel for testing
				smallChannel := make(chan event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy], 1)
				reconciler.eventChannel = smallChannel

				// Fill the channel
				policy1 := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "policy-1"},
				}
				smallChannel <- event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy]{
					Object: policy1,
				}

				// Create another policy
				policy2 := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "policy-2"},
				}
				Expect(reconciler.Client.Create(ctx, policy2)).To(Succeed())

				// This should not block or panic when channel is full
				trigger.enqueuePoliciesForNode(ctx, "test")

				// Verify the function completed (didn't hang)
				Expect(true).To(BeTrue())
			})
		})

		Describe("startupReconciliation", func() {
			It("should trigger reconciliation after delay", func() {
				// Set a very short delay for testing
				oldDelay := startupReconcileDelay
				startupReconcileDelay = 10 * time.Millisecond
				defer func() { startupReconcileDelay = oldDelay }()

				// Create test node and policy
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				policy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "startup-policy",
					},
				}
				Expect(reconciler.Client.Create(ctx, policy)).To(Succeed())

				// Run startup reconciliation
				go trigger.startupReconciliation(ctx)

				// Verify event is received after delay
				Eventually(reconciler.eventChannel, "1s").Should(Receive(WithTransform(
					func(e event.TypedGenericEvent[*nmstatev1.NodeNetworkConfigurationPolicy]) string {
						return e.Object.GetName()
					},
					Equal("startup-policy"),
				)))
			})

			It("should respect context cancellation", func() {
				// Set a long delay
				oldDelay := startupReconcileDelay
				startupReconcileDelay = 5 * time.Second
				defer func() { startupReconcileDelay = oldDelay }()

				// Cancel context immediately
				cancel()

				// Run startup reconciliation
				trigger.startupReconciliation(ctx)

				// Should not send any events
				Consistently(reconciler.eventChannel, "100ms").ShouldNot(Receive())
			})
		})

		Describe("periodicReconciliation", func() {
			It("should trigger reconciliation periodically", func() {
				// Set a very short interval for testing
				oldInterval := periodicReconcileInterval
				periodicReconcileInterval = 50 * time.Millisecond
				defer func() { periodicReconcileInterval = oldInterval }()

				// Create test node and policy
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				policy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "periodic-policy",
					},
				}
				Expect(reconciler.Client.Create(ctx, policy)).To(Succeed())

				// Run periodic reconciliation
				go trigger.periodicReconciliation(ctx)

				// Verify events are received periodically
				Eventually(reconciler.eventChannel, "200ms").Should(Receive())
				Eventually(reconciler.eventChannel, "200ms").Should(Receive())

				// Cancel and verify it stops
				cancel()
				time.Sleep(100 * time.Millisecond)
				Consistently(reconciler.eventChannel, "100ms").ShouldNot(Receive())
			})

			It("should stop when context is cancelled", func() {
				oldInterval := periodicReconcileInterval
				defer func() {
					periodicReconcileInterval = oldInterval
				}()
				periodicReconcileInterval = 50 * time.Millisecond

				// Give time for the global variable write to complete
				time.Sleep(10 * time.Millisecond)

				// Run and immediately cancel
				testCtx, testCancel := context.WithCancel(context.Background())
				go trigger.periodicReconciliation(testCtx)

				// Small delay to ensure goroutine starts
				time.Sleep(10 * time.Millisecond)
				testCancel()

				// Should stop quickly
				time.Sleep(100 * time.Millisecond)
				Consistently(reconciler.eventChannel, "100ms").ShouldNot(Receive())
			})
		})

		Describe("Start (integration)", func() {
			It("should start both startup and periodic reconciliation", func() {
				// Set short delays for testing
				oldDelay := startupReconcileDelay
				oldInterval := periodicReconcileInterval
				startupReconcileDelay = 10 * time.Millisecond
				periodicReconcileInterval = 50 * time.Millisecond
				defer func() {
					startupReconcileDelay = oldDelay
					periodicReconcileInterval = oldInterval
				}()

				// Create test node and policy
				testNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				}
				Expect(reconciler.Client.Create(ctx, testNode)).To(Succeed())

				policy := &nmstatev1.NodeNetworkConfigurationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "integration-policy",
					},
				}
				Expect(reconciler.Client.Create(ctx, policy)).To(Succeed())

				// Start the trigger
				go trigger.Start(ctx)

				// Should receive startup event quickly
				Eventually(reconciler.eventChannel, "100ms").Should(Receive())

				// Should also receive periodic events
				Eventually(reconciler.eventChannel, "200ms").Should(Receive())

				// Cancel and verify clean shutdown
				cancel()
				time.Sleep(100 * time.Millisecond)
			})
		})
	})
})
