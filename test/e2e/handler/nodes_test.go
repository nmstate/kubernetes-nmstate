package handler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/pkg/node"
)

var _ = Describe("[rfe_id:3503][crit:medium][vendor:cnv-qe@redhat.com][level:component][nns]Nodes", func() {
	Context("when are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(primaryNic))
			}
		})
		Context("and node network state is deleted", func() {
			BeforeEach(func() {
				deleteNodeNeworkStates()
			})
			It("should recreate it with currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(primaryNic))
				}
			})
		})
		Context("and new interface is configured", func() {
			expectedDummyName := "dummy0"

			BeforeEach(func() {
				createDummyConnectionAtNodes(expectedDummyName)
			})
			AfterEach(func() {
				deleteConnectionAndWait(nodes, expectedDummyName)
			})
			It("[test_id:3794]should update node network state with it", func() {
				for _, nodeName := range nodes {
					Eventually(func() []string {
						return interfacesNameForNode(nodeName)
					}, 2*node.NetworkStateRefresh, time.Second).Should(ContainElement(expectedDummyName))
				}
			})
		})
	})
})
