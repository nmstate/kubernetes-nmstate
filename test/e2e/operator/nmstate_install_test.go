package operator

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"
)

var (
	defaultNMState = nmstatev1beta1.NMState{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nmstate",
			Namespace: "nmstate",
		},
	}
	webhookKey    = types.NamespacedName{Namespace: "nmstate", Name: "nmstate-webhook"}
	handlerKey    = types.NamespacedName{Namespace: "nmstate", Name: "nmstate-handler"}
	handlerLabels = map[string]string{"component": "kubernetes-nmstate-handler"}
)

var _ = Describe("NMState operator", func() {
	Context("when installed for the first time", func() {
		BeforeEach(func() {
			installDefaultNMState()
		})
		It("should deploy daemonset and webhook deployment", func() {
			daemonset.GetEventually(handlerKey).Should(daemonset.BeReady())
			deployment.GetEventually(webhookKey).Should(deployment.BeReady())
		})
		AfterEach(func() {
			uninstallDefaultNMState()
		})
	})
	Context("when NMState is installed", func() {
		It("should list one NMState CR", func() {
			installDefaultNMState()
			daemonset.GetEventually(handlerKey).Should(daemonset.BeReady())
			ds, err := daemonset.GetList(handlerLabels)
			Expect(err).ToNot(HaveOccurred(), "List daemon sets in namespace nmstate succeeds")
			Expect(ds.Items).To(HaveLen(1), "and has only one item")
		})
		Context("and the CR is created with a wrong name", func() {
			var nmstate = defaultNMState
			nmstate.Name = "wrong-name"
			BeforeEach(func() {
				err := framework.Global.Client.Create(context.TODO(), &nmstate, &framework.CleanupOptions{})
				Expect(err).ToNot(HaveOccurred(), "NMState CR with incorrect name is created without error")
			})
			AfterEach(func() {
				err := framework.Global.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
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
				uninstallDefaultNMState()
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
})

func installNMState(nmstate nmstatev1beta1.NMState) {
	err := framework.Global.Client.Create(context.TODO(), &nmstate, &framework.CleanupOptions{})
	Expect(err).ToNot(HaveOccurred(), "NMState CR created without error")
}

func installDefaultNMState() {
	installNMState(defaultNMState)
}

func uninstallNMState(nmstate nmstatev1beta1.NMState) {
	err := framework.Global.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
	if !apierrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred(), "NMState CR successfully removed")
	}
}

func uninstallDefaultNMState() {
	uninstallNMState(defaultNMState)
}
