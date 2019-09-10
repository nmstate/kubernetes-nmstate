package helper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var (
	badYaml = nmstatev1alpha1.State("}")
	empty   = nmstatev1alpha1.State("")

	noBridges = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: bond1
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - eth1
      options:
        miimon: '120'
`)
	noBridgesUp = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: br1
    type: linux-bridge
    state: down
  - name: br2
    type: linux-bridge
    state: absent
`)

	bridgeWithNoPorts = nmstatev1alpha1.State(`interfaces:
  - name: br1
    type: linux-bridge
    state: up
`)

	someBridgesUp = nmstatev1alpha1.State(`interfaces:
  - name: eth1
    type: ethernet
    state: up
  - name: eth2
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
  - name: br2
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: eth2
  - name: br3
    type: linux-bridge
    state: down
  - name: br4
    type: linux-bridge
    state: absent
`)
)

var _ = Describe("Network desired state bridge parser", func() {
	var (
		obtainedBridgesAndPorts map[string][]string
		desiredState            nmstatev1alpha1.State
		err                     error
	)
	JustBeforeEach(func() {
		obtainedBridgesAndPorts, err = getBridgesUp(desiredState)
	})
	Context("when desired state is not a yaml", func() {
		BeforeEach(func() {
			desiredState = badYaml
		})
		It("should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
	Context("when desired state is empty", func() {
		BeforeEach(func() {
			desiredState = empty
		})
		It("should return empty map", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedBridgesAndPorts).To(BeEmpty())
		})
	})
	Context("when there is no bridges", func() {
		BeforeEach(func() {
			desiredState = noBridges
		})
		It("should return empty map", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedBridgesAndPorts).To(BeEmpty())
		})
	})
	Context("when there are no bridges up", func() {
		BeforeEach(func() {
			desiredState = noBridgesUp
		})
		It("should return empty map", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedBridgesAndPorts).To(BeEmpty())
		})
	})
	Context("when there are no ports in the bridge", func() {
		BeforeEach(func() {
			desiredState = bridgeWithNoPorts
		})
		It("should return empty map", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedBridgesAndPorts).To(BeEmpty())
		})
	})
	Context("when there are bridges up", func() {
		BeforeEach(func() {
			desiredState = someBridgesUp
		})
		It("should return the map with the bridges and ports", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(len(obtainedBridgesAndPorts)).To(Equal(2))
			ports, exist := obtainedBridgesAndPorts["br1"]
			Expect(exist).To(BeTrue())
			Expect(ports).To(ConsistOf("eth1"))
			ports, exist = obtainedBridgesAndPorts["br2"]
			Expect(exist).To(BeTrue())
			Expect(ports).To(ConsistOf("eth2"))
		})
	})
})
