package e2e

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NNS cleanup", func() {
	var nodeName types.NamespacedName

	BeforeEach(func() {
		nodeName = types.NamespacedName{Name: nodes[0]}

		By("Checking that NNS exists")
		Eventually(func() error {
			return framework.Global.Client.Get(context.TODO(), nodeName, &nmstatev1alpha1.NodeNetworkState{})
		}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	})

	Context("after node removal", func() {
		nodeToDelete := &corev1.Node{}

		BeforeEach(func() {
			By("Getting the node we want to delete")
			err := framework.Global.Client.Get(context.TODO(), nodeName, nodeToDelete)
			Expect(err).To(BeNil())

			By("Deleting the node")
			err = framework.Global.Client.Delete(context.TODO(), nodeToDelete)
			Expect(err).To(BeNil())
		})

		It("should remove NNS of that node", func() {
			Eventually(func() error {
				return framework.Global.Client.Get(context.TODO(), nodeName, &nmstatev1alpha1.NodeNetworkState{})
			}, ReadTimeout, ReadInterval).Should(HaveOccurred())
		})
	})
})
