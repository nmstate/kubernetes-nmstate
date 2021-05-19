package operator

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgoreporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

type operatorTestData struct {
	ns                                     string
	nmstate                                nmstatev1beta1.NMState
	webhookKey, handlerKey, certManagerKey types.NamespacedName
	handlerLabels                          map[string]string
}

func newOperatorTestData(ns string) operatorTestData {
	return operatorTestData{
		ns: ns,
		nmstate: nmstatev1beta1.NMState{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nmstate",
				Namespace: ns,
			},
		},
		webhookKey:     types.NamespacedName{Namespace: ns, Name: "nmstate-webhook"},
		handlerKey:     types.NamespacedName{Namespace: ns, Name: "nmstate-handler"},
		certManagerKey: types.NamespacedName{Namespace: ns, Name: "nmstate-cert-manager"},
	}
}

var (
	t               *testing.T
	nodes           []string
	startTime       time.Time
	defaultOperator = newOperatorTestData("nmstate")
	handlerLabels   = map[string]string{"component": "kubernetes-nmstate-handler"}
)

func TestE2E(t *testing.T) {
	testenv.TestMain()

	RegisterFailHandler(Fail)

	reporters := make([]Reporter, 0)
	reporters = append(reporters, knmstatereporter.New("test_logs/e2e/operator", testenv.OperatorNamespace, nodes))
	if ginkgoreporters.Polarion.Run {
		reporters = append(reporters, &ginkgoreporters.Polarion)
	}
	if ginkgoreporters.JunitOutput != "" {
		reporters = append(reporters, ginkgoreporters.NewJunitReporter())
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Operator E2E Test Suite", reporters)
}

var _ = BeforeSuite(func() {

	// Change to root directory some test expect that
	os.Chdir("../../../")

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testenv.Start()

	By("Getting node list from cluster")
	nodeList := corev1.NodeList{}
	err := testenv.Client.List(context.TODO(), &nodeList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, node := range nodeList.Items {
		nodes = append(nodes, node.Name)
	}
})

var _ = AfterSuite(func() {
	uninstallNMStateAndWaitForDeletion(defaultOperator)
})

func installNMState(nmstate nmstatev1beta1.NMState) {
	By(fmt.Sprintf("Creating NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Create(context.TODO(), &nmstate)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "NMState CR created without error")
}

func uninstallNMState(nmstate nmstatev1beta1.NMState) {
	By(fmt.Sprintf("Deleting NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
	Expect(err).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsNotFound, BeTrue())), "NMState CR successfully removed")
	eventuallyIsNotFound(types.NamespacedName{Name: nmstate.Name}, &nmstate, "should delete NMState CR")
}

func eventuallyIsNotFound(key types.NamespacedName, obj client.Object, msg string) {
	By(fmt.Sprintf("Wait for %+v deletion", key))
	EventuallyWithOffset(1, func() error {
		err := testenv.Client.Get(context.TODO(), key, obj)
		return err
	}, 120*time.Second, 1*time.Second).Should(WithTransform(apierrors.IsNotFound, BeTrue()), msg)
}

func eventuallyIsFound(key types.NamespacedName, obj client.Object, msg string) {
	By(fmt.Sprintf("Wait for %+v creation", key))
	EventuallyWithOffset(1, func() error {
		return testenv.Client.Get(context.TODO(), key, obj)
	}, 120*time.Second, 1*time.Second).Should(Succeed(), msg)
}

func uninstallNMStateAndWaitForDeletion(testData operatorTestData) {
	uninstallNMState(testData.nmstate)
	eventuallyOperandIsNotFound(testData)
}

func eventuallyOperandIsReady(testData operatorTestData) {
	eventuallyOperandIsFound(testData)
	By("Wait daemonset handler is ready")
	daemonset.GetEventually(testData.handlerKey).Should(daemonset.BeReady(), "should start handler daemonset")
	By("Wait deployment webhook is ready")
	deployment.GetEventually(testData.webhookKey).Should(deployment.BeReady(), "should start webhook deployment")
	By("Wait deployment cert-manager is ready")
	deployment.GetEventually(testData.certManagerKey).Should(deployment.BeReady(), "should start cert-manager deployment")
}

func eventuallyOperandIsNotFound(testData operatorTestData) {
	eventuallyIsNotFound(testData.handlerKey, &appsv1.DaemonSet{}, "should delete handler daemonset")
	eventuallyIsNotFound(testData.webhookKey, &appsv1.Deployment{}, "should delete webhook deployment")
	eventuallyIsNotFound(testData.certManagerKey, &appsv1.Deployment{}, "should delete cert-manager deployment")
	By("Wait for operand pods to terminate")
	Eventually(func() ([]corev1.Pod, error) {
		podList := corev1.PodList{}
		err := testenv.Client.List(context.TODO(), &podList, &client.ListOptions{Namespace: testData.ns, LabelSelector: labels.Set{"app": "kubernetes-nmstate"}.AsSelector()})
		return podList.Items, err
	}, 120*time.Second, time.Second).Should(BeEmpty(), "should terminate all the pods")

}

func eventuallyOperandIsFound(testData operatorTestData) {
	eventuallyIsFound(testData.handlerKey, &appsv1.DaemonSet{}, "should create handler daemonset")
	eventuallyIsFound(testData.webhookKey, &appsv1.Deployment{}, "should create webhook deployment")
	eventuallyIsFound(testData.certManagerKey, &appsv1.Deployment{}, "should create cert-manager deployment")
}
