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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

var (
	t              *testing.T
	nodes          []string
	startTime      time.Time
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
	uninstallNMState(defaultNMState)
})

func installNMState(nmstate nmstatev1beta1.NMState) {
	By(fmt.Sprintf("Creating NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Create(context.TODO(), &nmstate)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "NMState CR created without error")
}

func uninstallNMState(nmstate nmstatev1beta1.NMState) {
	By(fmt.Sprintf("Deleting NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
	if !apierrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred(), "NMState CR successfully removed")
	}
}
