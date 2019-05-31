package e2e

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	yaml "sigs.k8s.io/yaml"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

func findInterface(name string, interfaces nmstatev1.State) bool {
	//TODO: State is a interface{} we will have to convert it to json
	//      and look for the interface name
	return false
}

func waitInterface(namespace string, nodeName string, name string, mustExist bool) error {

	return wait.PollImmediate(5*time.Second, 50*time.Second, func() (bool, error) {
		var err error
		state := nmstatev1.NodeNetworkState{}
		err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName, Namespace: namespace}, &state)
		if err != nil {
			return false, err
		}
		exist := findInterface(name, state.Status.CurrentState)
		return exist == mustExist, nil
	})

}

func waitInterfaceCreated(namespace string, nodeName string, name string) error {
	return waitInterface(namespace, nodeName, name, true)
}

func waitInterfaceDeleted(namespace string, nodeName string, name string) error {
	return waitInterface(namespace, nodeName, name, false)
}

func prepare(t *testing.T) (*framework.TestCtx, string) {
	By("Initialize cluster resources")
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	Expect(err).ToNot(HaveOccurred())

	// get namespace
	namespace, err := ctx.GetNamespace()
	Expect(err).ToNot(HaveOccurred())

	// wait for memcached-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, framework.Global.KubeClient, namespace, "kubernetes-nmstate-manager", 1, time.Second*5, time.Second*30)
	Expect(err).ToNot(HaveOccurred())
	return ctx, namespace
}

func readStateFromFile(file string, namespace string) nmstatev1.NodeNetworkState {
	manifest, err := ioutil.ReadFile(createBridgeCr)
	Expect(err).ToNot(HaveOccurred())

	state := nmstatev1.NodeNetworkState{}
	err = yaml.Unmarshal(manifest, &state)
	Expect(err).ToNot(HaveOccurred())
	state.ObjectMeta.Namespace = namespace
	return state
}

func createStateFromFile(file string, namespace string, cleanupOptions *framework.CleanupOptions) {
	state := readStateFromFile(createBridgeCr, namespace)
	err := framework.Global.Client.Create(context.TODO(), &state, cleanupOptions)
	Expect(err).ToNot(HaveOccurred())
}

func updateStateSpecFromFile(file string, key types.NamespacedName) {
	state := nmstatev1.NodeNetworkState{}
	stateFromManifest := readStateFromFile(file, key.Namespace)
	err := framework.Global.Client.Get(context.TODO(), key, &state)
	Expect(err).ToNot(HaveOccurred())
	state.Spec = stateFromManifest.Spec
	err = framework.Global.Client.Update(context.TODO(), &state)
	Expect(err).ToNot(HaveOccurred())
}

func currentState(key types.NamespacedName) nmstatev1.State {
	state := nmstatev1.NodeNetworkState{}
	err := framework.Global.Client.Get(context.TODO(), key, &state)
	Expect(err).ToNot(HaveOccurred())
	return state.Status.CurrentState
}
