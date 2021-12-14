package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo"
)

type exampleSpec struct {
	name       string
	fileName   string
	policyName string
	ifaceNames []string
}

// This suite checks that all examples in our docs can be successfully applied.
// It only checks the top level API, hence it does not verify that the
// configuration is indeed applied on nodes. That should be tested by dedicated
// test suites for each feature.
var _ = Describe("[user-guide] Examples", func() {
	beforeTestIfaceExample := func(fileName string) {
		kubectlAndCheck("apply", "-f", fmt.Sprintf("docs/examples/%s", fileName))
	}

	testIfaceExample := func(policyName string) {
		kubectlAndCheck("wait", "nncp", policyName, "--for", "condition=Available", "--timeout", "2m")
	}

	afterIfaceExample := func(policyName string, ifaceNames []string) {
		deletePolicy(policyName)

		for _, ifaceName := range ifaceNames {
			updateDesiredStateAndWait(interfaceAbsent(ifaceName))
		}

		resetDesiredStateForNodes()
	}

	examples := []exampleSpec{
		{
			name:       "Ethernet",
			fileName:   "ethernet.yaml",
			policyName: "ethernet",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Linux bridge",
			fileName:   "linux-bridge.yaml",
			policyName: "linux-bridge",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "Linux bridge with custom vlan",
			fileName:   "linux-bridge-vlan.yaml",
			policyName: "linux-bridge-vlan",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "Detach bridge port and restore its configuration",
			fileName:   "detach-bridge-port-and-restore-eth.yaml",
			policyName: "detach-bridge-port-and-restore-eth",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "OVS bridge",
			fileName:   "ovs-bridge.yaml",
			policyName: "ovs-bridge",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "OVS bridge with interface",
			fileName:   "ovs-bridge-iface.yaml",
			policyName: "ovs-bridge-iface",
			ifaceNames: []string{"br1"},
		},
		{
			name:       "Linux bonding",
			fileName:   "bond.yaml",
			policyName: "bond",
			ifaceNames: []string{"bond0"},
		},
		{
			name:       "Linux bonding and VLAN",
			fileName:   "bond-vlan.yaml",
			policyName: "bond-vlan",
			ifaceNames: []string{"bond0.102", "bond0"},
		},
		{
			name:       "VLAN",
			fileName:   "vlan.yaml",
			policyName: "vlan",
			ifaceNames: []string{"eth1.102", "eth1"},
		},
		{
			name:       "DHCP",
			fileName:   "dhcp.yaml",
			policyName: "dhcp",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Static IP",
			fileName:   "static-ip.yaml",
			policyName: "static-ip",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "Route",
			fileName:   "route.yaml",
			policyName: "route",
			ifaceNames: []string{"eth1"},
		},
		{
			name:       "DNS",
			fileName:   "dns.yaml",
			policyName: "dns",
			ifaceNames: []string{},
		},
		{
			name:       "Worker selector",
			fileName:   "worker-selector.yaml",
			policyName: "worker-selector",
			ifaceNames: []string{"eth1"},
		},
	}

	for _, example := range examples {
		Context(example.name, func() {
			BeforeEach(func() {
				beforeTestIfaceExample(example.fileName)
			})

			AfterEach(func() {
				afterIfaceExample(example.policyName, example.ifaceNames)
			})

			It("should succeed applying the policy", func() {
				testIfaceExample(example.policyName)
			})
		})
	}
})
