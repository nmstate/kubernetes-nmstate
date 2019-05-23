package nmstate_tests

import (
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	yaml "github.com/ghodss/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
)

func waitBridge(nodeName string, name string, mustExist bool) error {

	return wait.Poll(5*time.Second, 50*time.Second, func() (bool, error) {
		var err error
		nns, err := nmstateNNSs.Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		_, exist := findInterfaceInfo(name, nns.Status.CurrentState.Interfaces)
		return exist == mustExist, nil
	})

}

func waitBridgeCreated(nodeName string, name string) error {
	return waitBridge(nodeName, name, true)
}

func waitBridgeDeleted(nodeName string, name string) error {
	return waitBridge(nodeName, name, false)
}

var _ = Describe("Linux Bridge", func() {
	Context("when exercising demo", func() {
		var expectedState *nmstatev1.NodeNetworkState
		bridgeName := "br1"
		createManifest := "create-br1-linux-bridge.yaml"
		deleteManifest := "delete-br1-linux-bridge.yaml"

		BeforeEach(func() {
			// TODO node01 is harcoded at linux-bridge.yaml
			By("Creating bridge with " + createManifest)
			manifest, err := ioutil.ReadFile(*manifests + createManifest)
			Expect(err).ToNot(HaveOccurred())

			state := nmstatev1.NodeNetworkState{}
			err = yaml.Unmarshal(manifest, &state)
			Expect(err).ToNot(HaveOccurred())

			expectedState, err = defaultNNSs.Create(&state)
			Expect(err).ToNot(HaveOccurred())

			err = waitBridgeCreated(firstNodeName, bridgeName)
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {

			// TODO node01 is harcoded at linux-bridge.yaml
			By("Deleting bridge with " + deleteManifest)
			manifest, err := ioutil.ReadFile(*manifests + deleteManifest)
			Expect(err).ToNot(HaveOccurred())

			nns := nmstatev1.NodeNetworkState{}
			err = yaml.Unmarshal(manifest, &nns)
			Expect(err).ToNot(HaveOccurred())

			deleteBr1, err := defaultNNSs.Get(firstNodeName, metav1.GetOptions{})
			deleteBr1.Spec = nns.Spec
			_, err = defaultNNSs.Update(deleteBr1)
			Expect(err).ToNot(HaveOccurred())

			// Wait for bridge to be removed
			err = waitBridgeDeleted(firstNodeName, bridgeName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have bridge configured as expected", func() {
			obtainedState, err := nmstateNNSs.Get(firstNodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			obtainedIf, _ := findInterfaceInfo(bridgeName, obtainedState.Status.CurrentState.Interfaces)
			expectedIf, _ := findInterfaceSpec(bridgeName, expectedState.Spec.DesiredState.Interfaces)
			Expect(obtainedIf.InterfaceSpec.Bridge).To(Equal(expectedIf.Bridge))
		})
	})
})
