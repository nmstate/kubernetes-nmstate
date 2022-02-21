package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("[nns] NNS Interface filter", func() {
	BeforeEach(func() {
		// Make sure NNSes are present
		for _, node := range nodes {
			key := types.NamespacedName{Name: node}
			_ = nodeNetworkState(key)
		}
	})
	It("should not log errors related to NNS interface filtering", func() {
		combinedHandlerLogs, err := cmd.Kubectl("logs", "-lname=nmstate-handler", "-n", "nmstate")
		Expect(err).ToNot(HaveOccurred())
		Expect(combinedHandlerLogs).ToNot(ContainSubstring(state.INTERFACE_FILTER))
	})
})
