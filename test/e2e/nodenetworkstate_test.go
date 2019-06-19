package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var _ = Describe("NodeNetworkState", func() {
	Context("when desiredState is configured", func() {
		Context("with a linux bridge", func() {
			var (
				br1Up = nmstatev1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: eth1
          stp-hairpin-mode: false
          stp-path-cost: 100
          stp-priority: 32
`)

				br1Absent = nmstatev1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: absent
`)
			)

			BeforeEach(func() {
				updateDesiredState(namespace, br1Up)
			})
			AfterEach(func() {
				updateDesiredState(namespace, br1Absent)
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					Eventually(func() []string {
						var currentStateYaml nmstatev1.State
						currentState(namespace, node, &currentStateYaml).ShouldNot(BeEmpty())

						interfaces := interfaces(currentStateYaml)
						Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

						return interfacesName(interfaces)
					}, ReadTimeout, ReadInterval).Should(ContainElement("br1"))
				}
			})
		})
	})
})
