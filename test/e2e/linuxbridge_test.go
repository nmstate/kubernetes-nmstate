package e2e

import (
	"context"
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
	createBridgeCr       = "deploy/crds/create-br1-linux-bridge.yaml"
	deleteBridgeCr       = "deploy/crds/delete-br1-linux-bridge.yaml"
	firstNodeName        = "node01" // TODO: Get it from cluster
	bridgeName           = "br1"
	namespace            string
	ctx                  *framework.TestCtx
)

var _ = Describe("Linux Bridge", func() {
	Context("when created", func() {

		var (
			state nmstatev1.NodeNetworkState
		)

		BeforeEach(func() {
			By("Initialize cluster resources")
			ctx = framework.NewTestCtx(t)
			err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
			Expect(err).ToNot(HaveOccurred())

			// get namespace
			namespace, err = ctx.GetNamespace()
			Expect(err).ToNot(HaveOccurred())

			// wait for memcached-operator to be ready
			err = e2eutil.WaitForOperatorDeployment(t, framework.Global.KubeClient, namespace, "memcached-operator", 1, time.Second*5, time.Second*30)
			Expect(err).ToNot(HaveOccurred())

			manifest, err := ioutil.ReadFile(createBridgeCr)
			Expect(err).ToNot(HaveOccurred())

			state = nmstatev1.NodeNetworkState{}
			err = yaml.Unmarshal(manifest, &state)
			Expect(err).ToNot(HaveOccurred())
			state.ObjectMeta.Namespace = namespace

		})

		It("should create the linux bridge successfully", func() {
			By("Creating the desiredState")
			err := framework.Global.Client.Create(context.TODO(), &state, &framework.CleanupOptions{
				TestContext:   ctx,
				Timeout:       cleanupTimeout,
				RetryInterval: cleanupRetryInterval})
			Expect(err).ToNot(HaveOccurred())

			By("Getting the currentState")
			obtainedState := nmstatev1.NodeNetworkState{}
			key := types.NamespacedName{Namespace: namespace, Name: firstNodeName}
			err = framework.Global.Client.Get(context.TODO(), key, &obtainedState)
			Expect(err).ToNot(HaveOccurred())
			currentState := obtainedState.CurrentState
			Expect(currentState).ToNot(BeEmpty())
		})

		AfterEach(func() {
			manifest, err := ioutil.ReadFile(deleteBridgeCr)
			Expect(err).ToNot(HaveOccurred())

			nns := nmstatev1.NodeNetworkState{}
			err = yaml.Unmarshal(manifest, &nns)
			Expect(err).ToNot(HaveOccurred())

			deleteBr1 := nmstatev1.NodeNetworkState{}
			err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Name: firstNodeName, Namespace: namespace}, &deleteBr1)
			Expect(err).ToNot(HaveOccurred())

			deleteBr1.Spec = nns.Spec
			err = framework.Global.Client.Update(context.TODO(), &deleteBr1)
			Expect(err).ToNot(HaveOccurred())

			// Wait for bridge to be removed
			err = waitInterfaceDeleted(namespace, firstNodeName, bridgeName)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
