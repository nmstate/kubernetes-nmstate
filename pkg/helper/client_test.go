package helper

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("FilterOut", func() {
	var (
		state, filteredState nmstatev1alpha1.State
	)

	Context("when given empty interface", func() {
		BeforeEach(func() {
			state = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
`)
			setEnvFilter("")
		})

		AfterEach(func() {
			interfacesFilterGlobIsSet = false
		})

		It("should return same state", func() {
			returnedState, err := filterOut(state)

			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(state))
		})
	})

	Context("when given invalid yaml", func() {
		BeforeEach(func() {
			state = nmstatev1alpha1.State(`invalid yaml`)
			setEnvFilter("{veth*}")
		})

		AfterEach(func() {
			interfacesFilterGlobIsSet = false
		})

		It("should return err", func() {
			_, err := filterOut(state)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("when given 2 interfaces and 1 is veth", func() {
		BeforeEach(func() {
			state = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
`)
			filteredState = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			setEnvFilter("{veth*}")
		})

		AfterEach(func() {
			interfacesFilterGlobIsSet = false
		})

		It("should return filtered 1 interface without veth", func() {
			returnedState, err := filterOut(state)

			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})

	Context("when given 3 interfaces and 2 are veths", func() {
		BeforeEach(func() {
			state = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
- name: vethjyuftrgv
  state: down
  type: ethernet
`)
			filteredState = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			setEnvFilter("{veth*}")
		})

		AfterEach(func() {
			interfacesFilterGlobIsSet = false
		})

		It("should return filtered 1 interface without veth", func() {
			returnedState, err := filterOut(state)

			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})

	Context("when given 3 interfaces, 1 is veth and 1 is vnet", func() {
		BeforeEach(func() {
			state = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
- name: vnet2b730a2b@if3
  state: down
  type: ethernet
`)
			filteredState = nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			setEnvFilter("{veth*,vnet*}")
		})

		AfterEach(func() {
			interfacesFilterGlobIsSet = false
		})

		It("should return filtered 1 interface without veth and vnet", func() {
			returnedState, err := filterOut(state)

			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})
})

func setEnvFilter(filter string) {
	os.Setenv("INTERFACES_FILTER", filter)
}
