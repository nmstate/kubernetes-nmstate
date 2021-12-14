package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("[nns] NNS Dependencies", func() {
	BeforeEach(func() {
		// Make sure NNSes are present
		for _, node := range nodes {
			key := types.NamespacedName{Name: node}
			_ = nodeNetworkState(key)
		}
	})

	It("should include versions of NNS dependencies", func() {
		for _, node := range nodes {
			key := types.NamespacedName{Name: node}
			status := nodeNetworkState(key).Status
			Expect(status.HostNetworkManagerVersion).ToNot(BeEmpty())
			Expect(status.HandlerNetworkManagerVersion).ToNot(BeEmpty())
			Expect(status.HandlerNmstateVersion).ToNot(BeEmpty())
		}
	})
})
