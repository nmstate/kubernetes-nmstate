package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NodeSelector", func() {
	br1Up := nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
      - name: eth1
`)
	br1Absent := nmstatev1alpha1.State(`interfaces:
- name: br1
  type: linux-bridge
  state: absent
`)
	nonexistentNodeSelector := map[string]string{"nonexistentKey": "nonexistentValue"}

	Context("when policy is set with node selector not matching any nodes", func() {
		BeforeEach(func() {
			setDesiredStateWithPolicyAndNodeSelector("br1", br1Up, nonexistentNodeSelector)
		})

		AfterEach(func() {
			setDesiredStateWithPolicy("br1", br1Absent)
			for _, node := range nodes {
				interfacesNameForNode(node).ShouldNot(ContainElement("br1"))
			}

			deletePolicy("br1")
		})

		It("should not update any nodes", func() {
			for _, node := range nodes {
				interfacesNameForNode(node).ShouldNot(ContainElement("br1"))
			}
		})

		Context("and we remove the node selector", func() {
			BeforeEach(func() {
				setDesiredStateWithPolicyAndNodeSelector("br1", br1Up, map[string]string{})
			})

			It("should update all nodes", func() {
				for _, node := range nodes {
					interfacesNameForNode(node).Should(ContainElement("br1"))
				}
			})
		})
	})
})
