package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeNetworkState", func() {
	Context("when vlan configured", func() {
		var (
			vlanId = "102"
		)

		BeforeEach(func() {
			updateDesiredState(ifaceUpWithVlanUp(firstSecondaryNic, vlanId))
			waitForAvailableTestPolicy()
		})
		AfterEach(func() {
			updateDesiredState(vlanAbsent(firstSecondaryNic, vlanId))
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})
		It("should have the vlan interface configured", func() {
			for _, node := range nodes {
				vlanForNodeInterfaceEventually(node, fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanId)).Should(Equal(vlanId))
			}
		})
	})
	//TODO: change static IP to DHCP once we have a DHCP server running on a VLAN.
	Context("when static address is configured on top of vlan interface", func() {
		var (
			ipAddressTemplate = "62.76.47.%d"
			vlanId            = "102"
		)
		BeforeEach(func() {
			updateDesiredState(ifaceUpWithVlanUp(firstSecondaryNic, vlanId))
			waitForAvailableTestPolicy()
			for index, node := range nodes {
				ipAddress := fmt.Sprintf(ipAddressTemplate, index)
				By(fmt.Sprintf("applying static IP %s on node %s", ipAddress, node))
				updateDesiredStateAtNode(node, vlanUpWithStaticIP(fmt.Sprintf("%s.%s", firstSecondaryNic, vlanId), ipAddress))
				waitForAvailableTestPolicy()
			}

		})

		AfterEach(func() {
			updateDesiredState(vlanAbsent(firstSecondaryNic, vlanId))
			waitForAvailableTestPolicy()
			resetDesiredStateForNodes()
		})

		It("should have the vlan interface configured and IP configured", func() {
			for index, node := range nodes {
				vlanForNodeInterfaceEventually(node, fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanId)).
					Should(Equal(vlanId))
				ipAddressForNodeInterfaceEventually(node, fmt.Sprintf(`%s.%s`, firstSecondaryNic, vlanId)).
					Should(Equal(fmt.Sprintf(ipAddressTemplate, index)))
			}
		})
	})
})
