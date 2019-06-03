package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	yaml "sigs.k8s.io/yaml"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		obtainedStruct   NodeNetworkState
		obtainedManifest []byte
		desiredState     = State(`
interfaces:
  - name: eth1
    type: ethernet
    state: up`)

		currentState = State(`
interfaces:
  - name: eth1
    type: ethernet
    state: down`)

		NNSManifest = `
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
currentState:
  interfaces:
    - name: eth1
      type: ethernet
      state: down

`
		NNSStruct = NodeNetworkState{
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
			CurrentState: currentState,
		}
	)

	BeforeEach(func() {
		err := yaml.Unmarshal([]byte(NNSManifest), &obtainedStruct)
		Expect(err).ToNot(HaveOccurred())

		obtainedManifest, err = yaml.Marshal(NNSStruct)
		Expect(err).ToNot(HaveOccurred())

	})
	Context("when unmarshal the result", func() {
		It("should be like the NNS struct", func() {
			// We cannot compare the whole structs since the raw yaml strings
			// are not going to match, so we have to match them field by field
			Expect(obtainedStruct.Spec.DesiredState).
				To(MatchYAML([]byte(NNSStruct.Spec.DesiredState)))

			Expect(obtainedStruct.CurrentState).
				To(MatchYAML([]byte(NNSStruct.CurrentState)))

			Expect(obtainedStruct.Spec.Managed).
				To(Equal(NNSStruct.Spec.Managed))

			Expect(obtainedStruct.Spec.NodeName).
				To(Equal(NNSStruct.Spec.NodeName))

			Expect(obtainedStruct.TypeMeta).To(Equal(NNSStruct.TypeMeta))

			Expect(obtainedStruct.ObjectMeta).To(Equal(NNSStruct.ObjectMeta))
		})
	})

	Context("when marshal the result", func() {
		It("should match the NNS manifest", func() {
			Expect(string(obtainedManifest)).To(MatchYAML(NNSManifest))
		})
	})

})
