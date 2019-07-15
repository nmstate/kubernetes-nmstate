package helper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("FilterOut", func() {
	Context("when given 2 interfaces and 1 is veth", func() {
		state := nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
- name: vethab6030bd
  state: down
  type: ethernet
`)
		returnedState := filterOut(state, "veth*")
		It("should return filtered 1 interface without veth", func() {
			filteredState := nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			Expect(returnedState).To(Equal(filteredState))
		})
	})

	Context("when given 3 interfaces and 2 are veths", func() {
		state := nmstatev1alpha1.State(`interfaces:
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
		returnedState := filterOut(state, "veth*")
		It("should return filtered 1 interface without veth", func() {
			filteredState := nmstatev1alpha1.State(`interfaces:
- name: eth1
  state: up
  type: ethernet
`)
			Expect(returnedState).To(Equal(filteredState))
		})
	})
})
