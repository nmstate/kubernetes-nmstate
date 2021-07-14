package handler

import (
	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	enactmentconditions "github.com/nmstate/kubernetes-nmstate/pkg/enactmentstatus/conditions"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createInterfaceWithInvalidName() nmstate.State {
	return nmstate.NewState(`interfaces:
  - name: eth666
    type: ethernet
    state: up`)
}

func createInterfaceWithInvalidField() nmstate.State {
	return nmstate.NewState(`interfaces:
  - name: eth1
    type: ethernet
    stasdte: up`)
}

func createInterfaceWithPingFail() nmstate.State {
	return nmstate.NewState(`interfaces:
  - name: eth0
    type: ethernet
    state: up
    ipv4:
      address:
      - ip: "192.168.45.33"
        prefix-length: 24
      dhcp: false
      enabled: true`)
}

func createPolicyAndWaitForEnactmentCondition(policy string, desiredState func() nmstate.State, nodeHostname string) {
	By("Creating the policy")
	err := setDesiredStateWithPolicyAndNodeSelector(policy, desiredState(), map[string]string{"kubernetes.io/hostname": nodeHostname})
	if err != nil {
		return
	}

	By("Waiting until the node becomes ready again")
	waitForNodesReady()

	By("Waiting for enactment to be failing")
	enactmentConditionsStatusEventually(nodes[0]).Should(matchConditionsFrom(enactmentconditions.SetFailedToConfigure))
}

var _ = Describe("NodeNetworkState", func() {
	var (
		defaultPolicy = "test-policy"

		messagesToRemove = [...]string{
			"DEBUG    Async action: Create checkpoint started",
			"DEBUG    Checkpoint None created for all devices",
			"Traceback (most recent call last):",
			"DEBUG    Nispor: current network state",
			"WARNING  libnm version",
			"rolling back desired state configuration: failed running probes after network changes: ",
			"failed running probe 'ping' with after network reconfiguration -> currentState:",
			"warnings.warn",
		}
	)

	Context("with invalid field", func() {
		var (
			messagesToKeep = [...]string{
				"libnmstate.error.NmstateVerificationError",
				"desired",
				"current",
				"difference",
			}
		)

		BeforeEach(func() {
			createPolicyAndWaitForEnactmentCondition(defaultPolicy, createInterfaceWithInvalidField, nodes[0])
		})

		It("should discard disarranged parts of the message", func() {
			for _, failMessage := range messagesToRemove {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).NotTo(ContainSubstring(failMessage))
			}
		})

		It("should keep desired parts of the message", func() {
			for _, failMessage := range messagesToKeep {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).To(ContainSubstring(failMessage))
			}
		})
	})

	Context("with invalid interface name", func() {
		var (
			messagesToKeep = [...]string{
				"libnmstate.error.NmstateLibnmError",
				"No suitable device found for this connection",
				"mismatching interface name",
			}
		)

		BeforeEach(func() {
			createPolicyAndWaitForEnactmentCondition(defaultPolicy, createInterfaceWithInvalidName, nodes[0])
		})

		It("should discard disarranged parts of the message", func() {
			for _, failMessage := range messagesToRemove {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).NotTo(ContainSubstring(failMessage))
			}
		})

		It("should keep desired parts of the message", func() {
			for _, validMessage := range messagesToKeep {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).To(ContainSubstring(validMessage))
			}
		})
	})

	Context("with ping fail", func() {
		var (
			messagesToKeep = [...]string{
				"libnmstate.error.NmstateValueError",
			}
		)

		BeforeEach(func() {
			createPolicyAndWaitForEnactmentCondition(defaultPolicy, createInterfaceWithPingFail, nodes[0])
		})

		AfterEach(func() {
			resetDesiredStateForNodes()
			By("Remove the policy")
			deletePolicy("test-policy")
		})

		It("should discard disarranged parts of the message", func() {

			for _, failMessage := range messagesToRemove {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).NotTo(ContainSubstring(failMessage))
			}
		})

		It("should keep desired parts of the message", func() {
			for _, validMessage := range messagesToKeep {
				Expect(enactmentConditionsStatus(nodes[0], defaultPolicy).Find(nmstate.NodeNetworkConfigurationEnactmentConditionFailing).Message).To(ContainSubstring(validMessage))
			}
		})
	})
})
