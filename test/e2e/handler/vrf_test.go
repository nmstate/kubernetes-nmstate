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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeNetworkState", func() {
	var (
		node string
	)

	BeforeEach(func() {
		node = nodes[0]
	})

	Context("when VRF configured", func() {
		var (
			vrfID            = "102"
			ipAddress        = "192.0.2.251"
			destIPAddress    = "198.51.100.0/24"
			prefixLen        = "24"
			nextHopIPAddress = "192.0.2.1"
		)

		BeforeEach(func() {
			updateDesiredStateAtNodeAndWait(node, vrfUp(vrfID, firstSecondaryNic))
			updateDesiredStateAtNodeAndWait(
				node,
				ipV4AddrAndRouteWithTableID(firstSecondaryNic, ipAddress, destIPAddress, prefixLen, nextHopIPAddress, vrfID),
			)
		})

		AfterEach(func() {
			updateDesiredStateAtNodeAndWait(node, vrfAbsent(vrfID))
			resetDesiredStateForNodes()
		})

		It("should have the VRF interface configured", func() {
			vrfForNodeInterfaceEventually(node, vrfID).Should(Equal(vrfID))
			ipAddressForNodeInterfaceEventually(node, firstSecondaryNic).Should(Equal(ipAddress))
			routeNextHopInterfaceWithTableID(node, destIPAddress, vrfID).Should(Equal(firstSecondaryNic))
		})
	})
})
