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

package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Simple OVS bridge", func() {
	Context("when desiredState is configured with an ovs bridge up", func() {
		BeforeEach(func() {
			updateDesiredStateAndWait(ovsBrUp(bridge1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(ovsBrAbsent(bridge1))
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
			updateDesiredStateAndWait(ovsbBrWithInternalInterface(bridge1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(ovsBrAbsent(bridge1))
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
