package operator

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("NMState operator", func() {
	Context("when installed for the first time", func() {
		BeforeEach(func() {
			installNMState(defaultNMState)
		})
		It("should deploy daemonset and webhook deployment", func() {
			daemonset.GetEventually(handlerKey).Should(daemonset.BeReady())
			deployment.GetEventually(webhookKey).Should(deployment.BeReady())
		})
		AfterEach(func() {
			uninstallNMState(defaultNMState)
		})
	})
	Context("when NMState is installed", func() {
		It("should list one NMState CR", func() {
			installNMState(defaultNMState)
			daemonset.GetEventually(handlerKey).Should(daemonset.BeReady())
			ds, err := daemonset.GetList(handlerLabels)
			Expect(err).ToNot(HaveOccurred(), "List daemon sets in namespace nmstate succeeds")
			Expect(ds.Items).To(HaveLen(1), "and has only one item")
		})
		Context("and another CR is created with a different name", func() {
			var nmstate = defaultNMState
			nmstate.Name = "different-name"
			BeforeEach(func() {
				err := testenv.Client.Create(context.TODO(), &nmstate)
				Expect(err).ToNot(HaveOccurred(), "NMState CR with different name is ignored")
			})
			AfterEach(func() {
				err := testenv.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), "NMState CR with incorrect name is removed without error")
			})
			It("should ignore it", func() {
				ds, err := daemonset.GetList(handlerLabels)
				Expect(err).ToNot(HaveOccurred(), "Daemon set list is retreieved without error")
				Expect(ds.Items).To(HaveLen(1), "and still only has one item")
			})
		})
		Context("and uninstalled", func() {
			BeforeEach(func() {
				uninstallNMState(defaultNMState)
			})
			It("should uninstall handler and webhook", func() {
				Eventually(func() bool {
					_, err := daemonset.Get(handlerKey)
					return apierrors.IsNotFound(err)
				}, 120*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprint("Daemon Set for NMState handler should be removed, but is not"))
				Eventually(func() bool {
					_, err := deployment.Get(webhookKey)
					return apierrors.IsNotFound(err)
				}, 120*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprint("Deployment for NMState webhook should be removed, but is not"))
			})
		})
	})
	Context("when another handler is installed with different namespace", func() {
		var (
			operatorNamespace = "nmstate-alt"
		)
		BeforeEach(func() {
			installNMState(defaultNMState)
			daemonset.GetEventually(handlerKey).Should(daemonset.BeReady())
			installOperator(operatorNamespace)
		})
		AfterEach(func() {
			uninstallNMState(defaultNMState)
			uninstallOperator(operatorNamespace)
			installOperator("nmstate")
		})
		It("should wait on the old one to be deleted", func() {
			By("Checking handler is locked")
			daemonset.GetConsistently(types.NamespacedName{Namespace: operatorNamespace, Name: "nmstate-handler"}).ShouldNot(daemonset.BeReady())
			uninstallOperator("nmstate")
			By("Checking handler is unlocked after deleting old one")
			daemonset.GetEventually(types.NamespacedName{Namespace: operatorNamespace, Name: "nmstate-handler"}).Should(daemonset.BeReady())
		})
	})
})

func installOperator(namespace string) error {
	By(fmt.Sprintf("Creating NMState operator with namespace '%s'", namespace))
	_, err := cmd.Run("make", false, fmt.Sprintf("OPERATOR_NAMESPACE=%s", namespace), fmt.Sprintf("HANDLER_NAMESPACE=%s", namespace), "IMAGE_REGISTRY=registry:5000", "manifests")
	Expect(err).ToNot(HaveOccurred())

	manifestsDir := "build/_output/manifests/"
	manifests := []string{"namespace.yaml", "service_account.yaml", "operator.yaml", "role.yaml", "role_binding.yaml"}
	for _, manifest := range manifests {
		_, err = cmd.Kubectl("apply", "-f", manifestsDir+manifest)
		Expect(err).ToNot(HaveOccurred())
	}
	deployment.GetEventually(types.NamespacedName{Namespace: namespace, Name: "nmstate-operator"}).Should(deployment.BeReady())

	return nil
}

func uninstallOperator(namespace string) {
	By(fmt.Sprintf("Deleting namespace '%s'", namespace))
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Expect(testenv.Client.Delete(context.TODO(), &ns)).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsNotFound, BeTrue())))
	Eventually(func() error {
		return testenv.Client.Get(context.TODO(), types.NamespacedName{Name: namespace}, &ns)
	}, 2*time.Minute, 5*time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(apierrors.IsNotFound, BeTrue())))
}
