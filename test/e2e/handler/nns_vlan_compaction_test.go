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
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("[nns] NNS VLAN compaction", func() {
	const (
		// vlanBaseID keeps the test VLANs away from the ones used by
		// policy based tests.
		vlanBaseID = 3000
		vlanCount  = 3

		// nnsMaxSizeBytesEnv is the operator environment variable that
		// configures the NodeNetworkState size budget on the handler.
		nnsMaxSizeBytesEnv = "NNS_MAX_SIZE_BYTES"
	)

	var (
		testNode  string
		vlanNames []string
	)

	vlanIPv4Address := func(vlanID int) string {
		return fmt.Sprintf("10.200.%d.1", vlanID-vlanBaseID)
	}

	createVlans := func() {
		vlanNames = []string{}
		for vlanID := vlanBaseID + 1; vlanID <= vlanBaseID+vlanCount; vlanID++ {
			vlanNames = append(vlanNames, fmt.Sprintf("%s.%d", firstSecondaryNic, vlanID))
			errs := createVlanConnection([]string{testNode}, firstSecondaryNic, vlanID, vlanIPv4Address(vlanID)+"/24")
			Expect(errs).To(HaveEach(BeNil()))
		}
		By("Waiting for the NodeNetworkState to report the VLANs")
		Eventually(func() []string {
			return interfacesNameForNode(testNode)
		}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElements(vlanNames))
	}

	deleteVlans := func() {
		for _, name := range vlanNames {
			deleteConnectionAndWait([]string{testNode}, name)
		}
	}

	interfaceFromNNS := func(g Gomega, name string) gjson.Result {
		path := fmt.Sprintf("interfaces.#(name==\"%s\")", name)
		iface := gjson.ParseBytes(currentStateJSON(testNode)).Get(path)
		g.Expect(iface.Exists()).To(BeTrue(), "interface %s should be reported in the NodeNetworkState", name)
		return iface
	}

	BeforeEach(func() {
		testNode = nodes[0]
	})

	Context("when the state fits within the size budget", func() {
		BeforeEach(func() {
			createVlans()
		})
		AfterEach(func() {
			deleteVlans()
		})
		It("should report the VLAN interfaces with all their fields", func() {
			Eventually(func(g Gomega) {
				for _, vlanName := range vlanNames {
					vlan := interfaceFromNNS(g, vlanName)
					g.Expect(vlan.Get("type").String()).To(Equal("vlan"))
					g.Expect(vlan.Get("state").String()).To(Equal("up"))
					g.Expect(vlan.Get("vlan.base-iface").String()).To(Equal(firstSecondaryNic))
					g.Expect(vlan.Get("mtu").Exists()).To(BeTrue(), "VLAN %s should keep the mtu field", vlanName)
					g.Expect(vlan.Get("mac-address").Exists()).To(BeTrue(), "VLAN %s should keep the mac-address field", vlanName)
				}
			}, ReadTimeout, ReadInterval).Should(Succeed())
		})
	})

	Context("when the state exceeds the size budget", func() {
		BeforeEach(func() {
			By("Configuring a size budget that no state can fit in")
			testenv.PatchOperatorEnv(map[string]string{nnsMaxSizeBytesEnv: "1"})

			createVlans()
		})
		AfterEach(func() {
			deleteVlans()

			By("Restoring the default size budget")
			testenv.PatchOperatorEnv(map[string]string{nnsMaxSizeBytesEnv: strconv.Itoa(state.DefaultMaxStateSizeBytes)})
		})
		It("should reduce VLAN interfaces to their identity and addresses and keep other interfaces intact", func() {
			By("Verifying the VLAN interfaces are compacted")
			Eventually(func(g Gomega) {
				for i, vlanName := range vlanNames {
					vlanID := vlanBaseID + 1 + i
					vlan := interfaceFromNNS(g, vlanName)
					g.Expect(vlan.Get("type").String()).To(Equal("vlan"))
					g.Expect(vlan.Get("state").String()).To(Equal("up"))
					g.Expect(vlan.Get("vlan.base-iface").String()).To(Equal(firstSecondaryNic))
					g.Expect(vlan.Get("vlan.id").Int()).To(Equal(int64(vlanID)))
					g.Expect(vlan.Get("ipv4.address.0.ip").String()).To(Equal(vlanIPv4Address(vlanID)))
					g.Expect(vlan.Get("mtu").Exists()).To(BeFalse(), "VLAN %s should not report the mtu field", vlanName)
					g.Expect(vlan.Get("mac-address").Exists()).To(BeFalse(), "VLAN %s should not report the mac-address field", vlanName)
					g.Expect(vlan.Get("vlan.protocol").Exists()).To(BeFalse(), "VLAN %s should not report the vlan protocol", vlanName)
				}
			}, ReadTimeout, ReadInterval).Should(Succeed())

			By("Verifying the other interfaces keep all their fields")
			for _, ifaceName := range []string{primaryNic, firstSecondaryNic} {
				iface := interfaceFromNNS(Default, ifaceName)
				Expect(iface.Get("mtu").Exists()).To(BeTrue(), "%s should keep the mtu field", ifaceName)
				Expect(iface.Get("mac-address").Exists()).To(BeTrue(), "%s should keep the mac-address field", ifaceName)
			}
		})
	})
})
