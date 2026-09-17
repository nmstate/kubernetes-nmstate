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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OVN-Kubernetes EgressIP IFA_PROTO", func() {
	const (
		dummyName      = "dummyeip"
		dummyIP        = "198.51.100.1"
		dummyPrefix    = "24"
		dummyCIDR      = dummyIP + "/" + dummyPrefix
		egressIPAddr   = "198.51.100.10"
		egressIPName   = "knmstate-ifa-proto-eip"
		testNamespace  = "knmstate-ifa-proto"
		testPodName    = "pause"
		ifaProtoLabel  = "knmstate.io/ifa-proto"
		ifaProtoValue  = "test"
		routeDest      = "203.0.113.0/24"
		routeNextHop   = "198.51.100.254"
		managedNMAddrs = dummyIP + "/" + dummyPrefix
	)

	var (
		node         string
		hadLabel     bool
		labeled      bool
		dummyCreated bool
		nsCreated    bool
		eipCreated   bool
	)

	BeforeEach(func() {
		skipUnlessEgressIPCRD()
		node = nodes[0]
		hadLabel = setEgressAssignableLabel(node)
		labeled = true

		updateDesiredStateAtNodeAndWait(node, dummyUpWithStaticIP(dummyName, dummyIP, dummyPrefix))
		dummyCreated = true
		ipAddressForNodeInterfaceEventually(node, dummyName).Should(Equal(dummyIP))
		waitForHostCIDROrSkip(node, dummyCIDR)

		labels := map[string]string{ifaProtoLabel: ifaProtoValue}
		createLabeledNamespace(testNamespace, labels)
		nsCreated = true
		createPausePod(testNamespace, testPodName, labels)

		createEgressIP(newEgressIP(
			egressIPName, egressIPAddr, ifaProtoLabel, ifaProtoValue, ifaProtoLabel, ifaProtoValue,
		))
		eipCreated = true
		waitForEgressIPAssignedOrSkip(egressIPName, egressIPAddr, node)

		info := waitForKernelAddrOnIface(node, egressIPAddr, dummyName)
		if info.Protocol == "" {
			Skip("IFA_PROTO not reported for EgressIP; kernel may be older than 5.18")
		}
		Expect(normalizeAddrProtocol(info.Protocol)).To(Equal(ifaProtOVNKHex))
	})

	AfterEach(func() {
		if node != "" {
			cleanupEgressIP(node, dummyName, egressIPName, egressIPAddr, managedNMAddrs, eipCreated)
			eipCreated = false
		}
		if dummyCreated {
			updateDesiredStateAtNodeAndWait(node, interfaceAbsent(dummyName))
			waitForInterfaceDeletion([]string{node}, dummyName)
			dummyCreated = false
		}
		deletePolicy(TestPolicy)
		if nsCreated {
			deleteNamespace(testNamespace)
			nsCreated = false
		}
		if labeled {
			restoreEgressAssignableLabel(node, hadLabel)
			labeled = false
		}
	})

	It("should not persist secondary-host EgressIP into the dummy NM profile", func() {
		updateDesiredStateAtNodeAndWait(node, routesOnlyIPv4OnIface(dummyName, routeDest, routeNextHop))

		Eventually(func(g Gomega) {
			assignedNode, assignedIP := egressIPAssignment(egressIPName)
			g.Expect(assignedNode).To(Equal(node), "EgressIP should stay assigned to the egress-assignable node")
			g.Expect(assignedIP).To(Equal(egressIPAddr))
		}, ReadTimeout, ReadInterval).Should(Succeed())
		info := waitForKernelAddrOnIface(node, egressIPAddr, dummyName)
		Expect(normalizeAddrProtocol(info.Protocol)).To(Equal(ifaProtOVNKHex))

		ipv4NM := nmDeviceAddresses(node, dummyName, "ipv4")
		Expect(ipv4NM).To(ContainSubstring(dummyIP))
		Expect(ipv4NM).NotTo(ContainSubstring(egressIPAddr))

		routeNextHopInterface(node, routeDest).Should(Equal(dummyName))
	})
})

func cleanupEgressIP(node, dummyName, eipName, eipAddr, managedAddrs string, eipCreated bool) {
	GinkgoHelper()
	if eipCreated {
		deleteEgressIP(eipName)
	}
	if addrs, ok := nmDeviceAddressesIfPresent(node, dummyName, "ipv4"); ok && strings.Contains(addrs, eipAddr) {
		restoreNMAddresses(node, dummyName, "ipv4", managedAddrs)
	}
	deleteKernelAddr(node, dummyName, eipAddr+"/32")
	deleteKernelAddr(node, dummyName, eipAddr+"/24")
	Eventually(func() string {
		info, _ := lookupKernelAddr(node, eipAddr)
		return info.Local
	}, ReadTimeout, ReadInterval).Should(BeEmpty())
}
