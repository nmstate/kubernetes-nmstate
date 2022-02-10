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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
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
		Entry("of a single node shoud deploy webhook replicas at the same node", controlPlaneTest{withMultiNode: false}),
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
				altOperator = NewOperatorTestData("nmstate-alt", manifestsDir, manifestFiles)
			)
			BeforeEach(func() {
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
			installNMState(defaultOperator.nmstate)
			eventuallyOperandIsReady(defaultOperator)
		})
		AfterEach(func() {
			uninstallNMStateAndWaitForDeletion(defaultOperator)
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
})

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
