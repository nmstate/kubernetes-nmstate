package e2e

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("NNS LastSuccessfulUpdateTime", func() {
	Context("when updating nns", func() {
		It("timestamp should be changed", func() {
			for _, node := range nodes {
				key := types.NamespacedName{Name: node}
				originalTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime

				configMap, err := framework.Global.KubeClient.CoreV1().ConfigMaps("nmstate").Get("nmstate-config", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				interval, err := strconv.Atoi(configMap.Data["node_network_state_refresh_interval"])
				Expect(err).ToNot(HaveOccurred())

				// Give enough time for the NNS to be updated (3 interval times)
				timeout := time.Duration(interval*3) * time.Second

				Eventually(func() time.Time {
					updatedTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
					return updatedTime.Time
				}, timeout, 1*time.Second).Should(BeTemporally(">", originalTime.Time))
			}
		})
	})
})
