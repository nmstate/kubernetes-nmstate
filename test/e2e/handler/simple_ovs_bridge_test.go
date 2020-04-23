package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//FIXME: OVS bridge functionality at nmstate is unstable
//       let's activate this again when that is fix
var _ = PDescribe("Simple OVS bridge [pending cause nmstate bug -> https://bugzilla.redhat.com/show_bug.cgi?id=1724901]", func() {
	Context("when desiredState is configured with an ovs bridge up", func() {
		BeforeEach(func() {
			updateDesiredState(ovsBrUp(bridge1))
			waitForAvailableTestPolicy()
		})
		AfterEach(func() {
			updateDesiredState(ovsBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			resetDesiredStateForNodes()
		})
		It("should have the ovs bridge at currentState", func() {
			for _, node := range nodes {
				interfacesForNode(node).Should(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", bridge1),
					HaveKeyWithValue("type", "ovs-bridge"),
					HaveKeyWithValue("state", "up"),
				)))
			}
		})
	})
	Context("when desiredState is configured with an ovs bridge with internal port up", func() {
		BeforeEach(func() {
			updateDesiredState(ovsbBrWithInternalInterface(bridge1))
			waitForAvailableTestPolicy()
		})
		AfterEach(func() {
			updateDesiredState(ovsBrAbsent(bridge1))
			waitForAvailableTestPolicy()
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			resetDesiredStateForNodes()
		})
		It("should have the ovs bridge at currentState", func() {
			for _, node := range nodes {
				interfacesForNode(node).Should(SatisfyAll(
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", bridge1),
						HaveKeyWithValue("type", "ovs-bridge"),
						HaveKeyWithValue("state", "up"),
					)),
					ContainElement(SatisfyAll(
						HaveKeyWithValue("name", "ovs0"),
						HaveKeyWithValue("type", "ovs-interface"),
						HaveKeyWithValue("state", "up"),
					)),
				))
			}
		})
	})
})
