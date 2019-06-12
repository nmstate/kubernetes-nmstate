package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	yaml "sigs.k8s.io/yaml"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var _ = Describe("Nodes", func() {
	Context("when nodes are up", func() {
		It("should have NodeNetworkState with currentState for each node", func() {
			for _, node := range nodes {
				key := types.NamespacedName{Namespace: namespace, Name: node}
				var currentStateYaml nmstatev1.State
				Eventually(func() nmstatev1.State {
					currentStateYaml = nodeNetworkState(key).Status.CurrentState
					return currentStateYaml
				}).ShouldNot(BeEmpty(), "Node %s should have currentState", node)

				By("unmarshal state yaml into unstructured golang")
				var currentState map[string]interface{}
				err := yaml.Unmarshal(currentStateYaml, &currentState)
				Expect(err).ToNot(HaveOccurred(), "Should parse correctly yaml: %s", currentStateYaml)

				interfaces := currentState["interfaces"].([]interface{})
				Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

				obtainedInterfaces := interfacesName(interfaces)
				Expect(obtainedInterfaces).To(SatisfyAll(
					ContainElement("eth0"),
					ContainElement("eth1"),
				))
			}
		})
	})
})
