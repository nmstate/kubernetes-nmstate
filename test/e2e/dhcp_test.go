package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("NNCP", func() {
	Context("when requesting an IP for a NIC from an available DHCP server", func() {
		ethUp := nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    type: ethernet
    state: up
    ipv4:
      enabled: true
      dhcp: true
`, *firstSecondaryNic))

		ethAbsent := nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: %s
    state: absent
`, *firstSecondaryNic))

		BeforeEach(func() {
			By("Removing existing NIC configuration")
			updateDesiredState(ethAbsent)
			waitForAvailableTestPolicy()

			By("Configuring NNCP for NIC with enabled DHCP")
			updateDesiredState(ethUp)
		})

		AfterEach(func() {
			By("Removing the NIC configuration")
			updateDesiredState(ethAbsent)
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})

		It("should successfully assign an IP to the interface and keep it for 30 seconds", func() {
			waitForAvailableTestPolicy()
			// TODO: iterate nodes, get the iface, make sure it has IP, it should be ok eventually and consistently
		})
	})

	Context("when requesting an IP from a DHCP server, but there is none available", func() {
		vlanUp := nmstatev1alpha1.NewState(fmt.Sprintf(`interfaces:
  - name: vlan100
    type: vlan
    state: up
    vlan:
      base-interface: %s
      id: 100
    ipv4:
      enabled: true
      dhcp: true
`, *firstSecondaryNic))

		vlanAbsent := nmstatev1alpha1.NewState(`interfaces:
  - name: vlan100
    state: absent
`)

		BeforeEach(func() {
			By("Configuring NNCP for VLAN with enabled DHCP")
			updateDesiredState(vlanUp)
		})

		AfterEach(func() {
			By("Removing the VLAN configuration")
			updateDesiredState(vlanAbsent)
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})

		It("should report failure and rollback to the original state", func() {
			// TODO: this should fail
			waitForAvailableTestPolicy()
		})
	})
})
