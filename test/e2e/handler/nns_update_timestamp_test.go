package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("[nns] NNS LastSuccessfulUpdateTime", func() {
	Context("when updating nns", func() {
		It("timestamp should be changed", func() {
			for _, node := range nodes {
				key := types.NamespacedName{Name: node}
				originalTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime

				// Give enough time for the NNS to be updated (3 interval times)
				timeout := time.Duration(5*3) * time.Second

				Eventually(func() time.Time {
					updatedTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
					return updatedTime.Time
				}, timeout, 1*time.Second).Should(BeTemporally(">", originalTime.Time))
			}
		})
	})
})
