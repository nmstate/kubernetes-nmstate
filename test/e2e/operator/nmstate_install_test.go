package operator

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("NMState operator", func() {
	Context("when installed for the first time", func() {
		BeforeEach(func() {
			By("Install NMState for the first time")
			installNMState(defaultOperator.nmstate)
		})
		It("should deploy a ready operand", func() {
			eventuallyOperandIsReady(defaultOperator)
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
	_, err := cmd.Run("make", false, fmt.Sprintf("OPERATOR_NAMESPACE=%s", operator.ns), fmt.Sprintf("HANDLER_NAMESPACE=%s", operator.ns), "manifests")
	Expect(err).ToNot(HaveOccurred())

	manifestsDir := "build/_output/manifests/"
	manifests := []string{"namespace.yaml", "service_account.yaml", "operator.yaml", "role.yaml", "role_binding.yaml"}
	for _, manifest := range manifests {
		_, err = cmd.Kubectl("apply", "-f", manifestsDir+manifest)
		Expect(err).ToNot(HaveOccurred())
	}
	cmd.Kubectl("apply", "-f", fmt.Sprintf("%s/scc.yaml", manifestsDir)) //ignore the error to be able to run the test against none OCP clusters as well

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
