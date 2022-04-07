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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"
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

			installNMState(defaultOperator.nmstate)
			defer uninstallNMStateAndWaitForDeletion(defaultOperator)
			eventuallyOperandIsReady(defaultOperator)

			By("Check webhook is distributed across control-plane nodes")
			podsShouldBeDistributedAtNodes(controlPlaneNodes(), client.MatchingLabels{"component": "kubernetes-nmstate-webhook"})
		},
		Entry("of a single node shoud deploy webhook replicas at the same node", controlPlaneTest{withMultiNode: false}),
		Entry("of two nodes should deploy webhook replicas at different nodes", controlPlaneTest{withMultiNode: true}),
	)
	Context("when installed for the first time", func() {
		BeforeEach(func() {
			By("Install NMState for the first time")
			installNMState(defaultOperator.nmstate)
		})
		AfterEach(func() {
			uninstallNMStateAndWaitForDeletion(defaultOperator)
		})
		Context("and another CR is created with different name", func() {
			var differentNMState = defaultOperator.nmstate
			differentNMState.Name = "different-name"
			BeforeEach(func() {
				eventuallyOperandIsReady(defaultOperator)
				installNMState(differentNMState)
			})
			It("should remove NMState with different name", func() {
				Eventually(func() error {
					return testenv.Client.Get(context.TODO(), types.NamespacedName{Name: differentNMState.Name}, &differentNMState)
				}, 120*time.Second, 1*time.Second).Should(WithTransform(apierrors.IsNotFound, BeTrue()))
			})

		})
		Context("and uninstalled", func() {
			BeforeEach(func() {
				uninstallNMState(defaultOperator.nmstate)
			})
			It("should uninstall handler and webhook", func() {
				eventuallyOperandIsNotFound(defaultOperator)
			})
		})
		Context("and another handler is installed with different namespace", func() {
			var (
				altOperator = newOperatorTestData("nmstate-alt")
			)
			BeforeEach(func() {
				By("Wait for operand to be ready")
				eventuallyOperandIsReady(defaultOperator)

				By("Install other operator at alternative namespace")
				installOperator(altOperator)
			})
			AfterEach(func() {
				uninstallOperator(altOperator)
				eventuallyOperandIsNotFound(altOperator)
				uninstallNMStateAndWaitForDeletion(defaultOperator)
				installOperator(defaultOperator)
			})
			It("should wait for defaultOperator handler to be deleted before deploying new altOperator handler", func() {
				By("Check alt handler has being created")
				Eventually(func() error {
					daemonSet := appsv1.DaemonSet{}
					return testenv.Client.Get(context.TODO(), altOperator.handlerKey, &daemonSet)
				}, 180*time.Second, 1*time.Second).Should(Succeed())

				By("Checking alt handler is locked")
				daemonset.GetConsistently(altOperator.handlerKey).ShouldNot(daemonset.BeReady())

				By("Uninstall default operator")
				uninstallOperator(defaultOperator)

				By("Checking alt handler is unlocked after deleting default one")
				daemonset.GetEventually(altOperator.handlerKey).Should(daemonset.BeReady())
			})
		})
	})
})

func installOperator(operator operatorTestData) error {
	By(fmt.Sprintf("Creating NMState operator with namespace '%s'", operator.ns))
	_, err := cmd.Run("make", false, fmt.Sprintf("OPERATOR_NAMESPACE=%s", operator.ns), fmt.Sprintf("HANDLER_NAMESPACE=%s", operator.ns), "IMAGE_REGISTRY=registry:5000", "manifests")
	Expect(err).ToNot(HaveOccurred())

	manifestsDir := "build/_output/manifests/"
	manifests := []string{"namespace.yaml", "service_account.yaml", "operator.yaml", "role.yaml", "role_binding.yaml"}
	for _, manifest := range manifests {
		_, err = cmd.Kubectl("apply", "-f", manifestsDir+manifest)
		Expect(err).ToNot(HaveOccurred())
	}
	deployment.GetEventually(types.NamespacedName{Namespace: operator.ns, Name: "nmstate-operator"}).Should(deployment.BeReady())

	return nil
}

func uninstallOperator(operator operatorTestData) {
	By(fmt.Sprintf("Deleting namespace '%s'", operator.ns))
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: operator.ns,
		},
	}
	Expect(testenv.Client.Delete(context.TODO(), &ns)).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsNotFound, BeTrue())))
	eventuallyIsNotFound(types.NamespacedName{Name: operator.ns}, &ns, "should delete the namespace")
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
