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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidwall/gjson"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/handler/linuxbridge"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
	"github.com/nmstate/kubernetes-nmstate/test/runner"
)

const ReadTimeout = 180 * time.Second
const ReadInterval = 1 * time.Second
const TestPolicy = "test-policy"

var (
	bridgeCounter  = 0
	bondCounter    = 0
	maxUnavailable = environment.GetVarWithDefault("NMSTATE_MAX_UNAVAILABLE", nmstatenode.DefaultMaxunavailable)
)

func Byf(message string, arguments ...interface{}) {
	By(fmt.Sprintf(message, arguments...))
}

func interfaceName(iface interface{}) string {
	name, hasName := iface.(map[string]interface{})["name"]
	Expect(hasName).
		To(
			BeTrue(),
			"should have name field in the interfaces, "+
				"https://github.com/nmstate/nmstate/blob/base/libnmstate/schemas/operational-state.yaml",
		)
	return name.(string)
}

func interfacesName(interfaces []interface{}) []string {
	var names []string
	for _, iface := range interfaces {
		names = append(names, interfaceName(iface))
	}
	return names
}

func interfaceByName(interfaces []interface{}, searchedName string) map[string]interface{} {
	var dummy map[string]interface{}
	for _, iface := range interfaces {
		if interfaceName(iface) == searchedName {
			return iface.(map[string]interface{})
		}
	}
	Fail(fmt.Sprintf("interface %s not found at %+v", searchedName, interfaces))
	return dummy
}

func setDesiredStateWithPolicyAndCaptureAndNodeSelector(
	name string,
	desiredState nmstate.State,
	capture map[string]string,
	nodeSelector map[string]string,
) error {
	policy := nmstatev1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	key := types.NamespacedName{Name: name}
	err := testenv.Client.Get(context.TODO(), key, &policy)
	policy.Spec.DesiredState = desiredState
	policy.Spec.Capture = capture
	policy.Spec.NodeSelector = nodeSelector
	maxUnavailableIntOrString := intstr.FromString(maxUnavailable)
	policy.Spec.MaxUnavailable = &maxUnavailableIntOrString
	if err != nil {
		if apierrors.IsNotFound(err) {
			return testenv.Client.Create(context.TODO(), &policy)
		}
		return err
	}
	err = testenv.Client.Update(context.TODO(), &policy)
	if err != nil {
		fmt.Println("Update error: " + err.Error())
	}
	return err
}

func setDesiredStateWithPolicyAndNodeSelector(name string, desiredState nmstate.State, nodeSelector map[string]string) error {
	return setDesiredStateWithPolicyAndCaptureAndNodeSelector(name, desiredState, nil, nodeSelector)
}

func setDesiredStateWithPolicyAndNodeSelectorEventually(name string, desiredState nmstate.State, nodeSelector map[string]string) {
	setDesiredStateWithPolicyAndCaptureAndNodeSelectorEventually(name, desiredState, nil, nodeSelector)
}

func setDesiredStateWithPolicyAndCaptureAndNodeSelectorEventually(
	name string,
	desiredState nmstate.State,
	capture map[string]string,
	nodeSelector map[string]string,
) {
	Eventually(func() error {
		return setDesiredStateWithPolicyAndCaptureAndNodeSelector(name, desiredState, capture, nodeSelector)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred(), fmt.Sprintf("Failed updating desired state : %s", desiredState))
	//FIXME: until we don't have webhook we have to wait for reconcile
	//       to start so we are sure that conditions are reset and we can
	//       check them correctly
	time.Sleep(1 * time.Second)
}

func setDesiredStateWithPolicyWithoutNodeSelector(name string, desiredState nmstate.State) {
	setDesiredStateWithPolicyAndNodeSelectorEventually(name, desiredState, map[string]string{})
}

func setDesiredStateWithPolicy(name string, desiredState nmstate.State) {
	runAtWorkers := map[string]string{"node-role.kubernetes.io/worker": ""}
	setDesiredStateWithPolicyAndNodeSelectorEventually(name, desiredState, runAtWorkers)
}

func setDesiredStateWithPolicyAndCapture(name string, desiredState nmstate.State, capture map[string]string) {
	runAtWorkers := map[string]string{"node-role.kubernetes.io/worker": ""}
	setDesiredStateWithPolicyAndCaptureAndNodeSelectorEventually(name, desiredState, capture, runAtWorkers)
}

func updateDesiredState(desiredState nmstate.State) {
	updateDesiredStateWithCapture(desiredState, nil)
}

func updateDesiredStateWithCapture(desiredState nmstate.State, capture map[string]string) {
	setDesiredStateWithPolicyAndCapture(TestPolicy, desiredState, capture)
}

func updateDesiredStateAndWait(desiredState nmstate.State) {
	updateDesiredStateWithCaptureAndWait(desiredState, nil)
}

func updateDesiredStateWithCaptureAndWait(desiredState nmstate.State, capture map[string]string) {
	updateDesiredStateWithCapture(desiredState, capture)
	policy.WaitForAvailableTestPolicy()
}

func updateDesiredStateAtNode(node string, desiredState nmstate.State) {
	updateDesiredStateWithCaptureAtNode(node, desiredState, nil)
}

func updateDesiredStateWithCaptureAtNode(node string, desiredState nmstate.State, capture map[string]string) {
	nodeSelector := map[string]string{"kubernetes.io/hostname": node}
	setDesiredStateWithPolicyAndCaptureAndNodeSelectorEventually(TestPolicy, desiredState, capture, nodeSelector)
}

func updateDesiredStateAtNodeAndWait(node string, desiredState nmstate.State) {
	updateDesiredStateWithCaptureAtNodeAndWait(node, desiredState, nil)
}

func updateDesiredStateWithCaptureAtNodeAndWait(node string, desiredState nmstate.State, capture map[string]string) {
	updateDesiredStateWithCaptureAtNode(node, desiredState, capture)
	policy.WaitForAvailableTestPolicy()
}

// TODO: After we implement policy delete (it will cleanUp desiredState) we have to remove this.
func resetDesiredStateForNodes() {
	By("Resetting nics state primary up and secondaries disable ipv4 and ipv6")
	updateDesiredState(resetPrimaryAndSecondaryNICs())
	defer deletePolicy(TestPolicy)
	policy.WaitForAvailableTestPolicy()
}

// TODO: After we implement policy delete (it will cleanUp desiredState) we have to remove this.
func resetDesiredStateForAllNodes() {
	By("Resetting nics state primary up and secondaries disable ipv4 and ipv6 at all nodes")
	setDesiredStateWithPolicyWithoutNodeSelector(TestPolicy, resetPrimaryAndSecondaryNICs())
	defer deletePolicy(TestPolicy)
	policy.WaitForAvailableTestPolicy()
}

func nodeNetworkState(key types.NamespacedName) nmstatev1beta1.NodeNetworkState {
	state := nmstatev1beta1.NodeNetworkState{}
	Eventually(func() error {
		return testenv.Client.Get(context.TODO(), key, &state)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return state
}

func nodeNetworkConfigurationPolicy(policyName string) nmstatev1.NodeNetworkConfigurationPolicy {
	key := types.NamespacedName{Name: policyName}
	policy := nmstatev1.NodeNetworkConfigurationPolicy{}
	EventuallyWithOffset(1, func() error {
		return testenv.Client.Get(context.TODO(), key, &policy)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return policy
}

func deleteNodeNeworkStates() {
	nodeNetworkStateList := &nmstatev1beta1.NodeNetworkStateList{}
	err := testenv.Client.List(context.TODO(), nodeNetworkStateList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	var deleteErrors []error
	for i := range nodeNetworkStateList.Items {
		deleteErrors = append(deleteErrors, testenv.Client.Delete(context.TODO(), &nodeNetworkStateList.Items[i]))
	}
	Expect(deleteErrors).ToNot(ContainElement(HaveOccurred()))
}

func deletePolicy(name string) {
	Byf("Deleting policy %s", name)
	policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	err := testenv.Client.Delete(context.TODO(), policy)
	if apierrors.IsNotFound(err) {
		return
	}
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Wait for policy to be removed
	EventuallyWithOffset(1, func() bool {
		err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: name}, &nmstatev1.NodeNetworkConfigurationPolicy{})
		return apierrors.IsNotFound(err)
	}, 60*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Policy %s not deleted", name))

	// Wait for enactments to be removed calculate timeout taking into account
	// the number of nodes, looks like it affect the time it takes to
	// delete enactments
	enactmentsDeleteTimeout := time.Duration(60+20*len(nodes)) * time.Second
	for _, node := range nodes {
		enactmentKey := nmstate.EnactmentKey(node, name)
		Eventually(func() bool {
			err := testenv.Client.Get(context.TODO(), enactmentKey, &nmstatev1beta1.NodeNetworkConfigurationEnactment{})
			// if we face an unexpected error do a failure since
			// we don't know if enactment was deleted
			if err != nil && !apierrors.IsNotFound(err) {
				Fail(fmt.Sprintf("Unexpected error waitting for enactment deletion: %v", err))
			}
			return apierrors.IsNotFound(err)
		}, enactmentsDeleteTimeout, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Enactment %s not deleted", enactmentKey.Name))
	}
}

func restartNodeWithoutWaiting(node string) {
	Byf("Restarting node %s", node)
	// Use halt so reboot command does not get stuck also
	// this command always fail since connection is closed
	// so let's not check err
	runner.RunAtNode(node, "sudo", "halt", "--reboot")
}

func waitForNodeToStart(node string) {
	Byf("Waiting till node %s is rebooted", node)
	// It will wait till uptime -p will return up that means that node was currently rebooted and is 0 min up
	Eventually(func() string {
		output, err := runner.RunAtNode(node, "uptime", "-p")
		if err != nil {
			return "not yet"
		}
		return output
	}, 300*time.Second, 5*time.Second).ShouldNot(Equal("up"), fmt.Sprintf("Node %s failed to start after reboot", node))
}

func createDummyConnection(nodesToModify []string, dummyName string) []error {
	Byf("Creating dummy %s", dummyName)
	_, errs := runner.RunAtNodes(
		nodesToModify,
		"sudo",
		"nmcli",
		"con",
		"add",
		"type",
		"dummy",
		"con-name",
		dummyName,
		"ifname",
		dummyName,
		"ip4",
		"192.169.1.50/24",
	)
	_, upErrs := runner.RunAtNodes(nodesToModify, "sudo", "nmcli", "con", "up", dummyName)
	errs = append(errs, upErrs...)
	return errs
}

func createDummyConnectionAtNodes(dummyName string) []error {
	return createDummyConnection(nodes, dummyName)
}

func createDummyConnectionAtAllNodes(dummyName string) []error {
	return createDummyConnection(allNodes, dummyName)
}

func deleteConnection(nodesToModify []string, name string) []error {
	Byf("Delete connection %s", name)
	_, errs := runner.RunAtNodes(nodesToModify, "sudo", "nmcli", "con", "delete", name)
	return errs
}

func deleteDevice(nodesToModify []string, name string) []error {
	Byf("Delete device %s  at nodes %v", name, nodesToModify)
	_, errs := runner.RunAtNodes(nodesToModify, "sudo", "nmcli", "device", "delete", name)
	return errs
}

func waitForInterfaceDeletion(nodesToCheck []string, interfaceName string) {
	for _, nodeName := range nodesToCheck {
		Eventually(func() []string {
			return interfacesNameForNode(nodeName)
		}, 2*nmstatenode.NetworkStateRefresh, time.Second).ShouldNot(ContainElement(interfaceName))
	}
}

func deleteConnectionAndWait(nodesToModify []string, interfaceName string) {
	deleteConnection(nodesToModify, interfaceName)
	deleteDevice(nodesToModify, interfaceName)
	waitForInterfaceDeletion(nodesToModify, interfaceName)
}

func interfaces(state nmstate.State) []interface{} {
	var stateUnstructured map[string]interface{}
	err := yaml.Unmarshal(state.Raw, &stateUnstructured)
	Expect(err).ToNot(HaveOccurred(), "Should parse correctly yaml: %s", state)
	interfaces := stateUnstructured["interfaces"].([]interface{})
	return interfaces
}

func currentState(node string, currentStateYaml *nmstate.State) AsyncAssertion {
	key := types.NamespacedName{Name: node}
	return Eventually(func() nmstate.RawState {
		*currentStateYaml = nodeNetworkState(key).Status.CurrentState
		return currentStateYaml.Raw
	}, ReadTimeout, ReadInterval)
}

func interfacesNameForNode(node string) []string {
	var currentStateYaml nmstate.State
	currentState(node, &currentStateYaml).ShouldNot(BeEmpty())

	interfaces := interfaces(currentStateYaml)
	Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

	return interfacesName(interfaces)
}

func interfacesNameForNodeEventually(node string) AsyncAssertion {
	return Eventually(func() []string {
		return interfacesNameForNode(node)
	}, ReadTimeout, ReadInterval)
}

func ipAddressForNodeInterfaceEventually(node, iface string) AsyncAssertion {
	return Eventually(func() string {
		return ipv4Address(node, iface)
	}, ReadTimeout, ReadInterval)
}

func ipV6AddressForNodeInterfaceEventually(node, iface string) AsyncAssertion {
	return Eventually(func() string {
		return ipv6Address(node, iface)
	}, ReadTimeout, ReadInterval)
}

func routeDestForNodeInterfaceEventually(node, destIP string) AsyncAssertion {
	return Eventually(func() string {
		return routeDest(node, destIP)
	}, ReadTimeout, ReadInterval)
}

func vlanForNodeInterfaceEventually(node, iface string) AsyncAssertion {
	return Eventually(func() string {
		return vlan(node, iface)
	}, ReadTimeout, ReadInterval)
}

// vrfForNodeInterfaceEventually asserts that VRF with vrfID is eventually created.
func vrfForNodeInterfaceEventually(node, vrfID string) AsyncAssertion {
	return Eventually(func() string {
		return vrf(node, vrfID)
	}, ReadTimeout, ReadInterval)
}

func interfacesForNode(node string) AsyncAssertion {
	return Eventually(func() []interface{} {
		var currentStateYaml nmstate.State
		currentState(node, &currentStateYaml).ShouldNot(BeEmpty())

		interfaces := interfaces(currentStateYaml)
		Expect(interfaces).ToNot(BeEmpty(), "Node %s should have network interfaces", node)

		return interfaces
	}, ReadTimeout, ReadInterval)
}

func bridgeVlansAtNode(node string) (string, error) {
	return runner.RunAtNode(node, "sudo", "bridge", "-j", "vlan", "show")
}

func getVLANFlagsEventually(node, connection string, vlan int) AsyncAssertion { //nolint:unparam
	Byf("Getting vlan filtering flags for node %s connection %s and vlan %d", node, connection, vlan)
	return Eventually(func() []string {
		bridgeVlans, err := bridgeVlansAtNode(node)
		if err != nil {
			return []string{}
		}

		if !gjson.Valid(bridgeVlans) {
			By("Getting vlan filtering from non-json output")
			// There is a bug [1] at centos8 and output is and invalid json
			// so it parses the non json output
			// [1] https://bugs.centos.org/view.php?id=16533
			output, err := cmd.Run("test/e2e/get-bridge-vlans-flags-el8.sh", false, node, connection, strconv.Itoa(vlan))
			Expect(err).ToNot(HaveOccurred())
			return strings.Split(output, " ")
		} else {
			By("Getting vlan filtering from json output")
			parsedBridgeVlans := gjson.Parse(bridgeVlans)

			gjsonExpression := linuxbridge.BuildGJsonExpression(bridgeVlans)
			vlanFlagsFilter := fmt.Sprintf(gjsonExpression+".flags", connection, vlan)

			vlanFlags := parsedBridgeVlans.Get(vlanFlagsFilter)
			if !vlanFlags.Exists() {
				return []string{}
			}

			matchingVLANFlags := []string{}
			for _, flag := range vlanFlags.Array() {
				matchingVLANFlags = append(matchingVLANFlags, flag.String())
			}
			return matchingVLANFlags
		}
	}, ReadTimeout, ReadInterval)
}

func hasVlans(node, connection string, minVlan, maxVlan int) AsyncAssertion { //nolint:unparam
	ExpectWithOffset(1, minVlan).To(BeNumerically(">", 0))
	ExpectWithOffset(1, maxVlan).To(BeNumerically(">", 0))
	ExpectWithOffset(1, maxVlan).To(BeNumerically(">=", minVlan))

	Byf("Check %s has %s with vlan filtering vids %d-%d", node, connection, minVlan, maxVlan)
	return Eventually(func() error {
		bridgeVlans, err := bridgeVlansAtNode(node)
		if err != nil {
			return err
		}
		if !gjson.Valid(bridgeVlans) {
			// There is a bug [1] at centos8 and output is and invalid json
			// so it parses the non json output
			// [1] https://bugs.centos.org/view.php?id=16533
			_, err := cmd.Run(
				"test/e2e/check-bridge-has-vlans-el8.sh",
				false,
				node,
				connection,
				strconv.Itoa(minVlan),
				strconv.Itoa(maxVlan),
			)
			if err != nil {
				return err
			}
		} else {
			parsedBridgeVlans := gjson.Parse(bridgeVlans)
			gjsonExpression := linuxbridge.BuildGJsonExpression(bridgeVlans)
			for expectedVlan := minVlan; expectedVlan <= maxVlan; expectedVlan++ {
				vlanByIDAndConection := fmt.Sprintf(gjsonExpression, connection, expectedVlan)
				if !parsedBridgeVlans.Get(vlanByIDAndConection).Exists() {
					return fmt.Errorf("bridge connection %s has no vlan %d, obtainedVlans: \n %s", connection, expectedVlan, bridgeVlans)
				}
			}
		}
		return nil
	}, ReadTimeout, ReadInterval)
}

func vlansCardinality(node, connection string) AsyncAssertion {
	Byf("Getting vlan cardinality for node %s connection %s", node, connection)
	return Eventually(func() (int, error) {
		bridgeVlans, err := bridgeVlansAtNode(node)
		if err != nil {
			return 0, err
		}

		return len(gjson.Parse(bridgeVlans).Get(connection).Array()), nil
	}, ReadTimeout, ReadInterval)
}

func bridgeDescription(node, bridgeName string) AsyncAssertion {
	return Eventually(func() (string, error) {
		return runner.RunAtNode(node, "sudo", "ip", "-d", "link", "show", "type", "bridge", bridgeName)
	}, ReadTimeout, ReadInterval)
}

func nextBridge() string {
	bridgeCounter++
	return fmt.Sprintf("br%d", bridgeCounter)
}

func nextBond() string {
	bondCounter++
	return fmt.Sprintf("bond%d", bondCounter)
}

func currentStateJSON(node string) []byte {
	key := types.NamespacedName{Name: node}
	currentState := nodeNetworkState(key).Status.CurrentState
	currentStateJSON, err := yaml.YAMLToJSON(currentState.Raw)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return currentStateJSON
}

func dhcpFlag(node, name string) bool {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.dhcp", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).Bool()
}

func autoDNS(node, name string) bool {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.auto-dns", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).Bool()
}

func ifaceInSlice(ifaceName string, names []string) bool {
	for _, name := range names {
		if ifaceName == name {
			return true
		}
	}
	return false
}

func nodeInterfacesState(node string, exclude []string) map[string]string {
	var currentStateYaml nmstate.State
	currentState(node, &currentStateYaml).ShouldNot(BeEmpty())
	return interfacesState(currentStateYaml, exclude)
}

// return a json with all node interfaces and their state e.g.
// {"cni0":"up","docker0":"up","eth0":"up","eth1":"down","eth2":"down","lo":"down"}
// use exclude to filter out interfaces you don't care about
func nodeInterfacesState(node string, exclude []string) map[string]string {
	var currentStateYaml nmstate.State
	currentState(node, &currentStateYaml).ShouldNot(BeEmpty())

	interfaces := interfaces(currentStateYaml)
	ifacesState := make(map[string]string)
	for _, iface := range interfaces {
		name := interfaceName(iface)
		if ifaceInSlice(name, exclude) {
			continue
		}
		state, hasState := iface.(map[string]interface{})["state"]
		if !hasState {
			state = "unknown"
		}
		if state == "ignore" {
			continue
		}
		ifacesState[name] = state.(string)
	}
	return ifacesState
}

func lldpNeighbors(node, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").lldp.neighbors", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func lldpEnabled(node, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").lldp.enabled", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func ipv4Address(node, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.address.0.ip", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func ipv6Address(node, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv6.address.0.ip", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func macAddress(node, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").mac-address", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func defaultRouteNextHopInterface(node string) AsyncAssertion {
	return Eventually(func() string {
		path := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-interface"
		return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
	}, 15*time.Second, 1*time.Second)
}

func routeDest(node, destIP string) string {
	path := fmt.Sprintf("routes.running.#(destination==%q)", destIP)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

// routeNextHopInterfaceWithTableID checks if a route with destIP exists in the default routing table.
func routeNextHopInterface(node, destIP string) AsyncAssertion {
	return routeNextHopInterfaceWithTableID(node, destIP, "")
}

// routeNextHopInterfaceWithTableID checks if a route with destIP exists in table tableID. If tableID is the empty
// string, use the default table-id (254).
func routeNextHopInterfaceWithTableID(node, destIP, tableID string) AsyncAssertion {
	if tableID == "" {
		tableID = "254"
	}
	return Eventually(func() string {
		path := fmt.Sprintf("routes.running.#(table-id==%s)#|#(destination==%q).next-hop-interface", tableID, destIP)
		return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
	}, 15*time.Second, 1*time.Second)
}

func vlan(node, iface string) string {
	vlanFilter := fmt.Sprintf("interfaces.#(name==\"%s\").vlan.id", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(vlanFilter).String()
}

// vrf verifies if the VRF with vrfID was created on node.
func vrf(node, vrfID string) string {
	vrfFilter := fmt.Sprintf("interfaces.#(name==vrf%s).vrf.route-table-id", vrfID)
	return gjson.ParseBytes(currentStateJSON(node)).Get(vrfFilter).String()
}

func kubectlAndCheck(command ...string) {
	out, err := cmd.Kubectl(command...)
	Expect(err).ShouldNot(HaveOccurred(), out)
}

func skipIfNotKubernetes() {
	provider := environment.GetVarWithDefault("KUBEVIRT_PROVIDER", "k8s")
	if !strings.Contains(provider, "k8s") {
		Skip("Tutorials use interface naming that is available only on Kubernetes providers")
	}
}

func maxUnavailableNodes() int {
	m, _ := nmstatenode.ScaledMaxUnavailableNodeCount(len(nodes), intstr.FromString(nmstatenode.DefaultMaxunavailable))
	return m
}

func dnsResolverServerForNodeEventually(node string) AsyncAssertion {
	return Eventually(func() []string {
		return dnsResolverForNode(node, "dns-resolver.running.server")
	}, ReadTimeout, ReadInterval)
}

func dnsResolverSearchForNodeEventually(node string) AsyncAssertion {
	return Eventually(func() []string {
		return dnsResolverForNode(node, "dns-resolver.running.search")
	}, ReadTimeout, ReadInterval)
}

func dnsResolverForNode(node, path string) []string {
	var arr []string

	elemList := gjson.ParseBytes(currentStateJSON(node)).Get(path).Array()
	for _, elem := range elemList {
		arr = append(arr, elem.String())
	}
	return arr
}

func ovnBridgeMappings(node string) string {
	result := gjson.ParseBytes(currentStateJSON(node)).Get("ovn.bridge-mappings")
	if result.String() == "" {
		return "null"
	}
	return result.String()
}
