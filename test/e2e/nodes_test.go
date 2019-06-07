package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	yaml "sigs.k8s.io/yaml"
)

var _ = Describe("Nodes", func() {
	Context("when nodes are up", func() {

		var (
			namespace string
			nodes     = []string{"node01"} // TODO: Get it from cluster
		)

		BeforeEach(func() {
			_, namespace = prepare(t)
		})

		AfterEach(func() {
			writePodsLogs(namespace, GinkgoWriter)
		})

		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				key := types.NamespacedName{Namespace: namespace, Name: node}
				nodeNetworkState := nodeNetworkState(key)
				currentStateYaml := nodeNetworkState.Status.CurrentState
				Expect(currentStateYaml).ToNot(BeEmpty())
				var currentState map[string][]map[string]interface{}
				err := yaml.Unmarshal(currentStateYaml, &currentState)
				Expect(err).ToNot(HaveOccurred())
				interfaces := currentState["interfaces"]
				Expect(interfaces).ToNot(BeEmpty())
				obtainedInterfaces := interfacesName(interfaces)
				Expect(obtainedInterfaces).To(SatisfyAll(
					ContainElement("lo"),
					ContainElement("cni0"),
					ContainElement("eth0"),
					ContainElement("eth1"),
					ContainElement("flannel.1"),
					ContainElement(ContainSubstring("veth")),
				))
			}
		})
	})
})
