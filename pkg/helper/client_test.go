package helper

import (
	"github.com/gobwas/glob"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("FilterOut", func() {
	var (
		state, filteredState nmstate.State
		interfacesFilterGlob glob.Glob
	)

	Context("when the filter is set to empty and there is a list of interfaces", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
`)
			interfacesFilterGlob = glob.MustCompile("")
		})

		It("should keep all interfaces intact", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(state))
		})
	})

	Context("when the filter is matching one of the interfaces in the list", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
`)
			filteredState = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			interfacesFilterGlob = glob.MustCompile("veth*")
		})

		It("should filter out matching interface and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})

	Context("when the filter is matching multiple interfaces in the list", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
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
			filteredState = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			interfacesFilterGlob = glob.MustCompile("veth*")
		})

		It("should filter out all matching interfaces and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})

	Context("when the filter is matching multiple prefixes", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`interfaces:
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
			filteredState = nmstate.NewState(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			interfacesFilterGlob = glob.MustCompile("{veth*,vnet*}")
		})

		It("it should filter out all interfaces matching any of these prefixes and keep the others", func() {
			returnedState, err := filterOut(state, interfacesFilterGlob)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(Equal(filteredState))
		})
	})
})
