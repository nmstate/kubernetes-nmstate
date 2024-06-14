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
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	policyconditions "github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
)

var _ = Describe("[nns] NNS OVN bridge mappings", func() {
	const (
		bridgeName  = "ovsbr1"
		networkName = "net1"
	)

	BeforeEach(func() {
		By("provisioning some bridge mappings ...")
		updateDesiredState(bridgeMappings(networkName, bridgeName))

		By("Check policy is at available state")
		policyconditions.WaitForAvailableTestPolicy()

		DeferCleanup(func() {
			By("resetting the bridge mappings ...")
			updateDesiredState(cleanBridgeMappings(networkName))

			By("Check policy is at available state")
			policyconditions.WaitForAvailableTestPolicy()
		})
	})

	It("are listed", func() {
		for _, node := range nodes {
			Expect(nodeBridgeMappings(node)).To(
				ContainElement(state.PhysicalNetworks{Name: networkName, Bridge: bridgeName}))
		}
	})
})

func nodeBridgeMappings(nodeName string) ([]state.PhysicalNetworks, error) {
	var physicalNetworks []state.PhysicalNetworks
	mappingsData := ovnBridgeMappings(nodeName)
	if err := json.Unmarshal([]byte(mappingsData), &physicalNetworks); err != nil {
		return nil, fmt.Errorf("failed to unmarshall bridge mappings for node %q: %w", nodeName, err)
	}
	return physicalNetworks, nil
}
