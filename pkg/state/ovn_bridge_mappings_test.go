/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("OVN bridge mappings", func() {
	var (
		state, filteredState nmstate.State
	)

	When("the OVN subtree is omitted", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
`)

			filteredState = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
`)
		})

		It("the OVN bridge mappings are not shown on the node network state", func() {
			returnedState, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	When("an empty OVN subtree is configured", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn: {}
`)

			filteredState = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn: {}
`)
		})

		It("the OVN subtree is reported, but mappings are not shown", func() {
			returnedState, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	When("mappings are provisioned", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn:
  bridge-mappings:
  - localnet: datanet
    bridge: ovs1
    state: present
`)
			filteredState = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn:
  bridge-mappings:
  - localnet: datanet
    bridge: ovs1
    state: present
`)
		})

		It("the OVN bridge mappings are listed in the node network state", func() {
			returnedState, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})

	When("an empty list of mappings is provisioned", func() {
		BeforeEach(func() {
			state = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn:
  bridge-mappings: []
`)
			filteredState = nmstate.NewState(`
interfaces: []
routes:
  config: []
  running: []
ovn: {}
`)
		})

		It("the OVN subtree (without mappings) is shown", func() {
			returnedState, err := filterOut(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedState).To(MatchYAML(filteredState))
		})
	})
})
