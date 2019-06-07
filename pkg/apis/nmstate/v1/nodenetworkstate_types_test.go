package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	yaml "sigs.k8s.io/yaml"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		desiredState = State(`
interfaces:
  - name: eth1
    type: ethernet
    state: up`)

		currentState = State(`
interfaces:
  - name: eth1
    type: ethernet
    state: down`)

		nnsManifest = `
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
  creationTimestamp: "1970-01-01T00:00:00Z"
spec:
  managed: true
  nodeName: node01
  desiredState:
    interfaces:
      - name: eth1
        type: ethernet
        state: up
status:
  currentState:
    interfaces:
      - name: eth1
        type: ethernet
        state: down

`
		nnsStruct = NodeNetworkState{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nmstate.io/v1",
				Kind:       "NodeNetworkState",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "node01",
				CreationTimestamp: metav1.Unix(0, 0),
			},
			Spec: NodeNetworkStateSpec{
				Managed:      true,
				NodeName:     "node01",
				DesiredState: desiredState,
			},
			Status: NodeNetworkStateStatus{
				CurrentState: currentState,
			},
		}
	)

	Context("when read NeworkNodeState struct from yaml", func() {

		var nodeNetworkStateStruct NodeNetworkState

		BeforeEach(func() {
			err := yaml.Unmarshal([]byte(nnsManifest), &nodeNetworkStateStruct)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succesfully parse desiredState yaml", func() {
			Expect(nodeNetworkStateStruct.Spec.DesiredState).To(MatchYAML([]byte(nnsStruct.Spec.DesiredState)))
		})
		It("should succesfully parse currentState yaml", func() {
			Expect(nodeNetworkStateStruct.Status.CurrentState).To(MatchYAML([]byte(nnsStruct.Status.CurrentState)))
		})
		It("should succesfully parse non state attributes", func() {
			Expect(nodeNetworkStateStruct.Spec.Managed).To(Equal(nnsStruct.Spec.Managed))
			Expect(nodeNetworkStateStruct.Spec.NodeName).To(Equal(nnsStruct.Spec.NodeName))
			Expect(nodeNetworkStateStruct.TypeMeta).To(Equal(nnsStruct.TypeMeta))
			Expect(nodeNetworkStateStruct.ObjectMeta).To(Equal(nnsStruct.ObjectMeta))
		})
	})

	Context("when reading NodeNetworkState struct from invalid yaml", func() {
		It("should return error", func() {
			err := yaml.Unmarshal([]byte("invalid yaml"), &NodeNetworkState{})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when write NetworkNodeState struct to yaml", func() {

		var nodeNetworkStateManifest []byte
		BeforeEach(func() {
			var err error
			nodeNetworkStateManifest, err = yaml.Marshal(nnsStruct)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should match the NodeNetworkState manifest", func() {
			Expect(string(nodeNetworkStateManifest)).To(MatchYAML(nnsManifest))
		})
	})

})
