package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NodeNetworkEnactment", func() {
	var (
		nncp = NodeNetworkConfigurationPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nmstate.io/v1beta1",
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
			nnce := NewEnactment(&node, nncp)
			desiredOnwerRefs := []metav1.OwnerReference{
				{Name: node.Name, Kind: "Node", APIVersion: "v1", UID: node.UID},
			}
			Expect(nnce.OwnerReferences).To(Equal(desiredOnwerRefs))
		})
	})

})
