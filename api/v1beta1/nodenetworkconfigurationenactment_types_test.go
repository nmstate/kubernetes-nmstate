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

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

var _ = Describe("NodeNetworkEnactment", func() {
	var (
		nncp = nmstatev1.NodeNetworkConfigurationPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nmstate.io/v1",
				Kind:       "NodeNetworkConfigurationPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy1",
				UID:  "12345",
			},
		}
		node = corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1",
				UID:  "54321",
			},
		}
	)

	Context("NewEnactment", func() {
		It("should have the node as the owner reference of the created enactment", func() {
			nnce := NewEnactment(&node, &nncp)
			desiredOwnerRefs := []metav1.OwnerReference{
				{Name: node.Name, Kind: "Node", APIVersion: "v1", UID: node.UID},
			}
			Expect(nnce.OwnerReferences).To(Equal(desiredOwnerRefs))
		})
		It("should have labels assocoating to the policy and the node", func() {
			nnce := NewEnactment(&node, &nncp)
			Expect(nnce.Labels).To(HaveKeyWithValue(shared.EnactmentPolicyLabel, nncp.Name))
			Expect(nnce.Labels).To(HaveKeyWithValue(shared.EnactmentNodeLabel, node.Name))
		})
	})

})
