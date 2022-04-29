/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	yaml "sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		currentState = nmstate.NewState(`
interfaces:
  - name: eth1
    type: ethernet
    state: down`)

		nnsManifest = `
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkState
metadata:
  name: node01
  creationTimestamp: "1970-01-01T00:00:00Z"
status:
  currentState:
    interfaces:
      - name: eth1
        type: ethernet
        state: down
  lastSuccessfulUpdateTime: "1970-01-01T00:00:00Z"

`
		nnsStruct = NodeNetworkState{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nmstate.io/v1alpha1",
				Kind:       "NodeNetworkState",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "node01",
				CreationTimestamp: metav1.Unix(0, 0),
			},
			Status: nmstate.NodeNetworkStateStatus{
				CurrentState:             currentState,
				LastSuccessfulUpdateTime: metav1.Unix(0, 0),
			},
		}
	)

	Context("when read NeworkNodeState struct from yaml", func() {

		var nodeNetworkStateStruct NodeNetworkState

		BeforeEach(func() {
			err := yaml.Unmarshal([]byte(nnsManifest), &nodeNetworkStateStruct)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should successfully parse currentState yaml", func() {
			Expect(string(nodeNetworkStateStruct.Status.CurrentState.Raw)).To(MatchYAML(string(nnsStruct.Status.CurrentState.Raw)))
		})
		It("should successfully parse non state attributes", func() {
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
