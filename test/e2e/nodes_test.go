package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Nodes", func() {
	Context("when nodes are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				key := types.NamespacedName{Namespace: namespace, Name: node}
				nodeNetworkState := nodeNetworkState(key)
				//TODO: exec nmstatectl is not in place let's just compare
				//      with the stuff we have harcoded there
				Expect(nodeNetworkState.Status.CurrentState).To(MatchYAML("interfaces: []"))
			}
		})
	})
})
