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

package operator

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"k8s.io/kubectl/pkg/drain"
)

const (
	verboseFlag      = "--v"
	verboseDebugFlag = "debug"
)

var _ = Describe("NMState operator", func() {
	type controlPlaneTest struct {
		withMultiNode bool
	}
	DescribeTable("for control-plane size",
		func(tc controlPlaneTest) {
			if isKubevirtciCluster() && tc.withMultiNode {
				kubevirtciReset := increaseKubevirtciControlPlane()
				defer kubevirtciReset()
			}
			if isKubevirtciCluster() && !tc.withMultiNode {
				uncordonNodeFunc := drainNode("node02")
				defer uncordonNodeFunc()
			}

			if tc.withMultiNode && len(controlPlaneNodes()) < 2 {
				Skip("cluster control-plane size should be > 1")
			}
			if !tc.withMultiNode && len(controlPlaneNodes()) > 1 {
				Skip("cluster control-plane size should be < 2")
			}

			InstallNMState(defaultOperator.Nmstate)
			defer UninstallNMStateAndWaitForDeletion(defaultOperator)
			EventuallyOperandIsReady(defaultOperator)

			By("Check webhook is distributed across control-plane nodes")
			podsShouldBeDistributedAtNodes(controlPlaneNodes(), client.MatchingLabels{"component": "kubernetes-nmstate-webhook"})
		},
		Entry("of a single node should deploy webhook replicas at the same node", controlPlaneTest{withMultiNode: false}),
		Entry("of two nodes should deploy webhook replicas at different nodes", controlPlaneTest{withMultiNode: true}),
	)
	Context("when installed for the first time", func() {
		BeforeEach(func() {
			By("Install NMState for the first time")
			InstallNMState(defaultOperator.Nmstate)
		})
		It("should deploy a ready operand", func() {
			EventuallyOperandIsReady(defaultOperator)
		})
		AfterEach(func() {
			UninstallNMStateAndWaitForDeletion(defaultOperator)
		})
		Context("and another CR is created with different name", func() {
			var differentNMState = defaultOperator.Nmstate
			differentNMState.Name = "different-name"
			BeforeEach(func() {
				EventuallyOperandIsReady(defaultOperator)
				InstallNMState(differentNMState)
			})
			It("should remove NMState with different name", func() {
				Eventually(func() error {
					return testenv.Client.Get(context.TODO(), types.NamespacedName{Name: differentNMState.Name}, &differentNMState)
				}, 120*time.Second, 1*time.Second).Should(WithTransform(apierrors.IsNotFound, BeTrue()))
			})

		})
		Context("and uninstalled", func() {
			BeforeEach(func() {
				UninstallNMState(defaultOperator.Nmstate)
			})
			It("should uninstall handler and webhook", func() {
				EventuallyOperandIsNotFound(defaultOperator)
			})
		})
		Context("and another handler is installed with different namespace", func() {
			var (
				altOperator TestData
			)
			BeforeEach(func() {
				altOperator = NewOperatorTestData(os.Getenv("HANDLER_NAMESPACE")+"-alt", manifestsDir, manifestFiles)
				By("Wait for operand to be ready")
				EventuallyOperandIsReady(defaultOperator)

				By("Install other operator at alternative namespace")
				InstallOperator(altOperator)
			})
			AfterEach(func() {
				UninstallOperator(altOperator)
				EventuallyOperandIsNotFound(altOperator)
				UninstallNMStateAndWaitForDeletion(defaultOperator)
				InstallOperator(defaultOperator)
			})
			It("should wait for defaultOperator handler to be deleted before deploying new altOperator handler", func() {
				By("Check alt handler has being created")
				Eventually(func() error {
					daemonSet := appsv1.DaemonSet{}
					return testenv.Client.Get(context.TODO(), altOperator.HandlerKey, &daemonSet)
				}, 180*time.Second, 1*time.Second).Should(Succeed())

				By("Checking alt handler is locked")
				daemonset.GetConsistently(altOperator.HandlerKey).ShouldNot(daemonset.BeReady())

				By("Uninstall default operator")
				UninstallOperator(defaultOperator)

				By("Checking alt handler is unlocked after deleting default one")
				daemonset.GetEventually(altOperator.HandlerKey).Should(daemonset.BeReady())
			})
		})
	})
	Context("when log level is configured", func() {
		Context("and deployed with debug mode", func() {
			BeforeEach(func() {
				debugNMState := defaultOperator.Nmstate
				debugNMState.Spec.LogLevel = shared.LogLevelDebug
				By("Install NMState with logLevel=debug")
				InstallNMState(debugNMState)
			})
			It("should deploy handler daemonset with verbose arguments", func() {
				EventuallyOperandIsReady(defaultOperator)

				By("Check handler daemonset has verbose arguments")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					// Check container args contain verbose flags
					args := daemonSet.Spec.Template.Spec.Containers[0].Args
					hasVFlag := false
					hasDebugFlag := false
					for i, arg := range args {
						if arg == verboseFlag && i+1 < len(args) && args[i+1] == verboseDebugFlag {
							hasVFlag = true
							hasDebugFlag = true
							break
						}
					}
					return hasVFlag && hasDebugFlag
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), "handler daemonset should have verbose arguments")

				By("Check handler daemonset livenessProbe uses verbose flag")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					// Check livenessProbe command contains verbose flag
					probe := daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe
					if probe == nil || probe.Exec == nil {
						return false
					}

					return slices.Contains(probe.Exec.Command, "nmstatectl show -vv 2>&1")
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), "handler daemonset livenessProbe should use verbose flag")
			})
			AfterEach(func() {
				UninstallNMStateAndWaitForDeletion(defaultOperator)
			})
		})

		Context("and deployed with info mode then updated to debug", func() {
			BeforeEach(func() {
				infoNMState := defaultOperator.Nmstate
				infoNMState.Spec.LogLevel = shared.LogLevelInfo
				By("Install NMState with logLevel=info")
				InstallNMState(infoNMState)
				EventuallyOperandIsReady(defaultOperator)
			})
			It("should update handler daemonset when log level changes", func() {
				By("Verify initial info mode deployment")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					// Check container args do NOT contain verbose flags
					args := daemonSet.Spec.Template.Spec.Containers[0].Args
					for i, arg := range args {
						if arg == verboseFlag && i+1 < len(args) && args[i+1] == verboseDebugFlag {
							return false // Should not have verbose flags
						}
					}
					return true
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), "handler should not have verbose arguments in info mode")

				By("Verify initial info mode livenessProbe does not use verbose flag")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					probe := daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe
					if probe == nil || probe.Exec == nil {
						return false
					}

					for _, cmd := range probe.Exec.Command {
						if cmd == "nmstatectl show -vv 2>&1" {
							return false // Should not have verbose flag in info mode
						}
						if cmd == "nmstatectl show  2>&1" {
							return true // Should have plain nmstatectl show command
						}
					}
					return false
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), "handler daemonset livenessProbe should not use verbose flag in info mode")

				By("Update NMState CR to debug mode")
				nmstateObj := defaultOperator.Nmstate
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: nmstateObj.Name}, &nmstateObj)
					if err != nil {
						return err
					}
					nmstateObj.Spec.LogLevel = shared.LogLevelDebug
					return testenv.Client.Update(context.TODO(), &nmstateObj)
				})
				Expect(err).ToNot(HaveOccurred(), "should update NMState CR to debug mode")

				By("Verify handler daemonset gets updated with verbose arguments")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					// Check container args contain verbose flags
					args := daemonSet.Spec.Template.Spec.Containers[0].Args
					hasVFlag := false
					hasDebugFlag := false
					for i, arg := range args {
						if arg == verboseFlag && i+1 < len(args) && args[i+1] == verboseDebugFlag {
							hasVFlag = true
							hasDebugFlag = true
							break
						}
					}
					return hasVFlag && hasDebugFlag
				}, 120*time.Second, 2*time.Second).Should(BeTrue(), "handler daemonset should be updated with verbose arguments")

				By("Verify livenessProbe is updated with verbose flag")
				Eventually(func() bool {
					daemonSet := appsv1.DaemonSet{}
					err := testenv.Client.Get(context.TODO(), defaultOperator.HandlerKey, &daemonSet)
					if err != nil {
						return false
					}

					probe := daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe
					if probe == nil || probe.Exec == nil {
						return false
					}

					for _, cmd := range probe.Exec.Command {
						if cmd == "nmstatectl show -vv 2>&1" {
							return true
						}
					}
					return false
				}, 120*time.Second, 2*time.Second).Should(BeTrue(), "handler daemonset livenessProbe should be updated with verbose flag")
			})
			AfterEach(func() {
				UninstallNMStateAndWaitForDeletion(defaultOperator)
			})
		})
	})
	Context("when cluser-reader exists", func() {
		const (
			clusterReaderRoleName = "cluster-reader"
			testUserNamespace     = "default"
			serviceAccountName    = "test-user"
			testUserBindingName   = "test-user-binding"
		)

		var clusterReaderCreated bool

		BeforeEach(func() {
			err := createClusterReaderCR(clusterReaderRoleName)
			Expect(err).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsAlreadyExists, BeTrue())))
			if err == nil {
				clusterReaderCreated = true
			}

			Expect(createTestUserSA(testUserNamespace, serviceAccountName)).To(Succeed(),
				"should success creating a serviceaccount")
			Expect(createTestUserCRB(testUserBindingName, testUserNamespace, serviceAccountName, clusterReaderRoleName)).To(Succeed(),
				"should success creating a clusterrolebinding")

			By("Install NMState for the first time")
			InstallNMState(defaultOperator.Nmstate)
			EventuallyOperandIsReady(defaultOperator)
		})
		AfterEach(func() {
			UninstallNMStateAndWaitForDeletion(defaultOperator)
		})
		AfterEach(func() {
			Expect(deleteTestUserCRB(testUserBindingName)).To(Succeed())
		})
		AfterEach(func() {
			Expect(deleteTestUserSA(testUserNamespace, serviceAccountName)).To(Succeed())
		})
		AfterEach(func() {
			if clusterReaderCreated {
				Expect(deleteClusterReaderCR(clusterReaderRoleName)).To(Succeed())
			}
		})

		It("should be able to read NMState resources with cluster-reader user", func() {
			Eventually(func() error {
				_, err := cmd.Kubectl("get", "nns", fmt.Sprintf("--as=system:serviceaccount:%s:%s", testUserNamespace, serviceAccountName))
				return err
			}, 10*time.Second, time.Second).Should(Succeed())
		})
	})

	Context("when checking NMState CRD status", func() {
		BeforeEach(func() {
			By("Install NMState for the first time")
			InstallNMState(defaultOperator.Nmstate)
		})
		AfterEach(func() {
			UninstallNMStateAndWaitForDeletion(defaultOperator)
		})

		It("should report Available condition when all components are ready", func() {
			By("Wait for operand to be ready")
			EventuallyOperandIsReady(defaultOperator)

			By("Check NMState status conditions")
			Eventually(func() shared.ConditionList {
				return getNMStateStatus(defaultOperator.Nmstate.Name).Conditions
			}, 60*time.Second, 2*time.Second).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(shared.NmstateConditionAvailable),
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal(shared.NmstateSuccessfullyDeployed),
			})), "should have Available condition set to True")

			By("Check Progressing condition is False when ready")
			Eventually(func() shared.ConditionList {
				return getNMStateStatus(defaultOperator.Nmstate.Name).Conditions
			}, 60*time.Second, 2*time.Second).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(shared.NmstateConditionProgressing),
				"Status": Equal(corev1.ConditionFalse),
				"Reason": Equal(shared.NmstateSuccessfullyDeployed),
			})), "should have Progressing condition set to False")

			By("Check Degraded condition is False when ready")
			Eventually(func() shared.ConditionList {
				return getNMStateStatus(defaultOperator.Nmstate.Name).Conditions
			}, 60*time.Second, 2*time.Second).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(shared.NmstateConditionDegraded),
				"Status": Equal(corev1.ConditionFalse),
				"Reason": Equal(shared.NmstateSuccessfullyDeployed),
			})), "should have Degraded condition set to False")
		})

		It("should report Progressing condition during deployment", func() {
			By("Check Progressing condition appears during installation")
			Eventually(func() shared.ConditionList {
				return getNMStateStatus(defaultOperator.Nmstate.Name).Conditions
			}, 120*time.Second, 1*time.Second).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(shared.NmstateConditionProgressing),
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal(shared.NmstateDeploying),
			})), "should have Progressing condition set to True during deployment")

			By("Check Available condition is False during deployment")
			Consistently(func() shared.ConditionList {
				conditions := getNMStateStatus(defaultOperator.Nmstate.Name).Conditions
				progressingCondition := conditions.Find(shared.NmstateConditionProgressing)
				if progressingCondition != nil && progressingCondition.Status == corev1.ConditionTrue {
					return conditions
				}
				return shared.ConditionList{} // Return empty if not progressing
			}, 10*time.Second, 1*time.Second).Should(SatisfyAny(
				BeEmpty(), // Not progressing anymore
				ContainElement(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(shared.NmstateConditionAvailable),
					"Status": Equal(corev1.ConditionFalse),
					"Reason": Equal(shared.NmstateDeploying),
				})),
			), "should have Available condition set to False while progressing")

			By("Wait for final ready state")
			EventuallyOperandIsReady(defaultOperator)
		})

		It("should maintain consistent condition transitions", func() {
			By("Wait for operand to be ready")
			EventuallyOperandIsReady(defaultOperator)

			By("Verify all three conditions are present and consistent")
			Eventually(func() bool {
				conditions := getNMStateStatus(defaultOperator.Nmstate.Name).Conditions

				availableCondition := conditions.Find(shared.NmstateConditionAvailable)
				progressingCondition := conditions.Find(shared.NmstateConditionProgressing)
				degradedCondition := conditions.Find(shared.NmstateConditionDegraded)

				// All conditions should be present
				if availableCondition == nil || progressingCondition == nil || degradedCondition == nil {
					return false
				}

				// In success state: Available=True, Progressing=False, Degraded=False
				// All should have the same reason
				if availableCondition.Status == corev1.ConditionTrue &&
					progressingCondition.Status == corev1.ConditionFalse &&
					degradedCondition.Status == corev1.ConditionFalse {

					return availableCondition.Reason == shared.NmstateSuccessfullyDeployed &&
						progressingCondition.Reason == shared.NmstateSuccessfullyDeployed &&
						degradedCondition.Reason == shared.NmstateSuccessfullyDeployed
				}

				return false
			}, 60*time.Second, 2*time.Second).Should(BeTrue(), "all conditions should be consistent in success state")
		})

		It("should have proper status subresource fields", func() {
			By("Wait for operand to be ready")
			EventuallyOperandIsReady(defaultOperator)

			By("Check status subresource contains expected fields")
			Eventually(func() bool {
				status := getNMStateStatus(defaultOperator.Nmstate.Name)

				// Should have conditions
				if len(status.Conditions) == 0 {
					return false
				}

				// Each condition should have required fields
				for _, condition := range status.Conditions {
					if condition.Type == "" || condition.Status == "" {
						return false
					}
					if condition.LastTransitionTime.IsZero() {
						return false
					}
					if condition.LastHeartbeatTime.IsZero() {
						return false
					}
				}

				return true
			}, 60*time.Second, 2*time.Second).Should(BeTrue(), "status should contain properly formatted conditions")
		})

		It("should handle kubectl status commands correctly", func() {
			By("Wait for operand to be ready")
			EventuallyOperandIsReady(defaultOperator)

			By("Check kubectl get nmstates shows status columns")
			Eventually(func() string {
				output, err := cmd.Kubectl("get", "nmstates")
				if err != nil {
					return ""
				}
				return output
			}, 30*time.Second, 2*time.Second).Should(SatisfyAll(
				ContainSubstring("STATUS"),
				ContainSubstring("REASON"),
			), "should be able to get nmstates with status columns")

			By("Check kubectl describe shows status conditions")
			Eventually(func() string {
				output, err := cmd.Kubectl("describe", "nmstate", defaultOperator.Nmstate.Name)
				if err != nil {
					return ""
				}
				return output
			}, 30*time.Second, 2*time.Second).Should(ContainSubstring("Conditions:"), "should show conditions in describe output")
		})

	})
})

func getNMStateStatus(name string) nmstatev1.NMStateStatus {
	nmstateInstance := &nmstatev1.NMState{}
	key := types.NamespacedName{Name: name}
	err := testenv.Client.Get(context.TODO(), key, nmstateInstance)
	if err != nil {
		return nmstatev1.NMStateStatus{}
	}
	return nmstateInstance.Status
}

func drainNode(nodeName string) func() {
	node := &corev1.Node{}
	drainer := drain.Helper{
		Ctx:                 context.TODO(),
		Client:              testenv.KubeClient,
		IgnoreAllDaemonSets: true,
		DeleteEmptyDirData:  true,
		Out:                 GinkgoWriter,
		ErrOut:              GinkgoWriter,
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := testenv.Client.Get(context.TODO(), client.ObjectKey{Name: nodeName}, node)
		if err != nil {
			return err
		}

		By(fmt.Sprintf("Cordon kubevirtci cluster node %s", node.Name))
		err = drain.RunCordonOrUncordon(&drainer, node, true)
		if err != nil {
			return err
		}

		By(fmt.Sprintf("Drain kubevirtci cluster node %s", node.Name)) //not really needed but to be sure to remove running pods from node...
		return drain.RunNodeDrain(&drainer, node.Name)
	})
	Expect(err).ToNot(HaveOccurred())

	return func() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := testenv.Client.Get(context.TODO(), client.ObjectKey{Name: nodeName}, node)
			if err != nil {
				return err
			}

			By(fmt.Sprintf("Uncordon kubevirtci cluster node %s", node.Name))
			return drain.RunCordonOrUncordon(&drainer, node, false)
		})
		Expect(err).ToNot(HaveOccurred())
	}
}

func increaseKubevirtciControlPlane() func() {
	secondNodeName := "node02"
	node := &corev1.Node{}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := testenv.Client.Get(context.TODO(), client.ObjectKey{Name: secondNodeName}, node)
		if err != nil {
			return err
		}
		By(fmt.Sprintf("Configure kubevirtci cluster node %s as control plane", node.Name))
		node.Labels["node-role.kubernetes.io/control-plane"] = ""
		node.Labels["node-role.kubernetes.io/master"] = ""
		return testenv.Client.Update(context.TODO(), node)
	})
	Expect(err).ToNot(HaveOccurred())
	return func() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := testenv.Client.Get(context.TODO(), client.ObjectKey{Name: secondNodeName}, node)
			if err != nil {
				return err
			}
			By(fmt.Sprintf("Configure kubevirtci cluster node %s as non control plane", node.Name))
			delete(node.Labels, "node-role.kubernetes.io/control-plane")
			delete(node.Labels, "node-role.kubernetes.io/master")
			return testenv.Client.Update(context.TODO(), node)
		})
		Expect(err).ToNot(HaveOccurred())
	}
}

func createClusterReaderCR(clusterReaderRoleName string) error {
	clusterReader := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterReaderRoleName,
		},
		AggregationRule: &rbacv1.AggregationRule{
			ClusterRoleSelectors: []metav1.LabelSelector{
				{
					MatchLabels: map[string]string{"rbac.authorization.k8s.io/aggregate-to-cluster-reader": "true"},
				},
			},
		},
	}
	return testenv.Client.Create(context.TODO(), &clusterReader)
}

func createTestUserSA(testUserNamespace, serviceAccountName string) error {
	testUserSA := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testUserNamespace,
			Name:      serviceAccountName,
		},
	}
	return testenv.Client.Create(context.TODO(), &testUserSA)
}

func createTestUserCRB(testUserBindingName, testUserNamespace, serviceAccountName, clusterReaderRoleName string) error {
	testUserBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: testUserNamespace,
				Name:      serviceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterReaderRoleName,
			APIGroup: rbacv1.GroupName,
		},
	}
	return testenv.Client.Create(context.TODO(), &testUserBinding)
}

func deleteClusterReaderCR(clusterReaderRoleName string) error {
	clusterReader := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterReaderRoleName,
		},
	}
	return testenv.Client.Delete(context.TODO(), &clusterReader)
}

func deleteTestUserSA(testUserNamespace, serviceAccountName string) error {
	testUserSA := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testUserNamespace,
			Name:      serviceAccountName,
		},
	}
	return testenv.Client.Delete(context.TODO(), &testUserSA)
}

func deleteTestUserCRB(testUserBindingName string) error {
	testUserBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserBindingName,
		},
	}
	return testenv.Client.Delete(context.TODO(), &testUserBinding)
}
