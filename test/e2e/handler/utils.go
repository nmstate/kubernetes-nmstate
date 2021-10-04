package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tidwall/gjson"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	yaml "sigs.k8s.io/yaml"

	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/handler/linuxbridge"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
	runner "github.com/nmstate/kubernetes-nmstate/test/runner"
)

const ReadTimeout = 180 * time.Second
const ReadInterval = 1 * time.Second
const TestPolicy = "test-policy"

var (
	bridgeCounter  = 0
	bondConunter   = 0
	maxUnavailable = environment.GetVarWithDefault("NMSTATE_MAX_UNAVAILABLE", nmstatenode.DEFAULT_MAXUNAVAILABLE)
)

func Byf(message string, arguments ...interface{}) {
	By(fmt.Sprintf(message, arguments...))
}

func interfacesName(interfaces []interface{}) []string {
	var names []string
	for _, iface := range interfaces {
		name, hasName := iface.(map[string]interface{})["name"]
		Expect(hasName).To(BeTrue(), "should have name field in the interfaces, https://github.com/nmstate/nmstate/blob/base/libnmstate/schemas/operational-state.yaml")
		names = append(names, name.(string))
	}
	return names
}

func interfaceByName(interfaces []interface{}, searchedName string) map[string]interface{} {
	var dummy map[string]interface{}
	for _, iface := range interfaces {
		name, hasName := iface.(map[string]interface{})["name"]
		Expect(hasName).To(BeTrue(), "should have name field in the interfaces, https://github.com/nmstate/nmstate/blob/base/libnmstate/schemas/operational-state.yaml")
		if name == searchedName {
			return iface.(map[string]interface{})
		}
	}
	Fail(fmt.Sprintf("interface %s not found at %+v", searchedName, interfaces))
	return dummy
}

func setDesiredStateWithPolicyAndNodeSelector(name string, desiredState nmstate.State, nodeSelector map[string]string) error {
	policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	key := types.NamespacedName{Name: name}
	err := testenv.Client.Get(context.TODO(), key, &policy)
	policy.Spec.DesiredState = desiredState
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

func setDesiredStateWithPolicyAndNodeSelectorEventually(name string, desiredState nmstate.State, nodeSelector map[string]string) {
	Eventually(func() error {
		return setDesiredStateWithPolicyAndNodeSelector(name, desiredState, nodeSelector)
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

func updateDesiredState(desiredState nmstate.State) {
	setDesiredStateWithPolicy(TestPolicy, desiredState)
}

func updateDesiredStateAndWait(desiredState nmstate.State) {
	updateDesiredState(desiredState)
	waitForAvailableTestPolicy()
}

func updateDesiredStateAtNode(node string, desiredState nmstate.State) {
	nodeSelector := map[string]string{"kubernetes.io/hostname": node}
	setDesiredStateWithPolicyAndNodeSelectorEventually(TestPolicy, desiredState, nodeSelector)
}

func updateDesiredStateAtNodeAndWait(node string, desiredState nmstate.State) {
	updateDesiredStateAtNode(node, desiredState)
	waitForAvailableTestPolicy()
}

// TODO: After we implement policy delete (it will cleanUp desiredState) we have
//       to remove this
func resetDesiredStateForNodes() {
	By("Resetting nics state primary up and secondaries down")
	updateDesiredState(resetPrimaryAndSecondaryNICs())
	waitForAvailableTestPolicy()
	deletePolicy(TestPolicy)
}

func nodeNetworkState(key types.NamespacedName) nmstatev1beta1.NodeNetworkState {
	state := nmstatev1beta1.NodeNetworkState{}
	Eventually(func() error {
		return testenv.Client.Get(context.TODO(), key, &state)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return state
}

func nodeNetworkConfigurationPolicy(policyName string) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	key := types.NamespacedName{Name: policyName}
	policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
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
	for _, nodeNetworkState := range nodeNetworkStateList.Items {
		deleteErrors = append(deleteErrors, testenv.Client.Delete(context.TODO(), &nodeNetworkState))
	}
	Expect(deleteErrors).ToNot(ContainElement(HaveOccurred()))
}

func deletePolicy(name string) {
	Byf("Deleting policy %s", name)
	policy := &nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	err := testenv.Client.Delete(context.TODO(), policy)
	if apierrors.IsNotFound(err) {
		return
	}
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Wait for policy to be removed
	EventuallyWithOffset(1, func() bool {
		err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: name}, &nmstatev1beta1.NodeNetworkConfigurationPolicy{})
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

func restartNode(node string) error {
	restartNodeWithoutWaiting(node)
	return waitFotNodeToStart(node)
}

func restartNodeWithoutWaiting(node string) {
	Byf("Restarting node %s", node)
	// Use halt so reboot command does not get stuck also
	// this command always fail since connection is closed
	// so let's not check err
	runner.RunAtNode(node, "sudo", "halt", "--reboot")
}

func waitFotNodeToStart(node string) error {
	Byf("Waiting till node %s is rebooted", node)
	// It will wait till uptime -p will return up that means that node was currently rebooted and is 0 min up
	Eventually(func() string {
		output, err := runner.RunAtNode(node, "uptime", "-p")
		if err != nil {
			return "not yet"
		}
		return output
	}, 300*time.Second, 5*time.Second).ShouldNot(Equal("up"), fmt.Sprintf("Node %s failed to start after reboot", node))
	return nil
}

func deleteBridgeAtNodes(bridgeName string, ports ...string) []error {
	Byf("Delete bridge %s", bridgeName)
	_, errs := runner.RunAtNodes(nodes, "sudo", "ip", "link", "del", bridgeName)
	for _, portName := range ports {
		_, portErrors := runner.RunAtNodes(nodes, "sudo", "nmcli", "con", "delete", bridgeName+"-"+portName)
		errs = append(errs, portErrors...)
	}
	return errs
}

func createDummyConnection(nodesToModify []string, dummyName string) []error {
	Byf("Creating dummy %s", dummyName)
	_, errs := runner.RunAtNodes(nodesToModify, "sudo", "nmcli", "con", "add", "type", "dummy", "con-name", dummyName, "ifname", dummyName, "ip4", "192.169.1.50/24")
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

func ipAddressForNodeInterfaceEventually(node string, iface string) AsyncAssertion {
	return Eventually(func() string {
		return ipv4Address(node, iface)
	}, ReadTimeout, ReadInterval)
}

func vlanForNodeInterfaceEventually(node string, iface string) AsyncAssertion {
	return Eventually(func() string {
		return vlan(node, iface)
	}, ReadTimeout, ReadInterval)
}

func interfacesNameForNodeConsistently(node string) AsyncAssertion {
	return Consistently(func() []string {
		return interfacesNameForNode(node)
	}, 5*time.Second, 1*time.Second)
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

func waitForNodeNetworkStateUpdate(node string) {
	now := time.Now()
	EventuallyWithOffset(1, func() time.Time {
		key := types.NamespacedName{Name: node}
		nnsUpdateTime := nodeNetworkState(key).Status.LastSuccessfulUpdateTime
		return nnsUpdateTime.Time
	}, 4*time.Minute, 5*time.Second).Should(BeTemporally(">=", now), fmt.Sprintf("Node %s should have a fresh nns)", node))

}

func toUnstructured(y string) interface{} {
	var u interface{}
	err := yaml.Unmarshal([]byte(y), &u)
	Expect(err).ToNot(HaveOccurred())
	return u
}

func bridgeVlansAtNode(node string) (string, error) {
	return runner.RunAtNode(node, "sudo", "bridge", "-j", "vlan", "show")
}

func getVLANFlagsEventually(node string, connection string, vlan int) AsyncAssertion {
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
			return strings.Split(string(output), " ")
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

func hasVlans(node string, connection string, minVlan int, maxVlan int) AsyncAssertion {

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
			_, err := cmd.Run("test/e2e/check-bridge-has-vlans-el8.sh", false, node, connection, strconv.Itoa(minVlan), strconv.Itoa(maxVlan))
			if err != nil {
				return err
			}
		} else {
			parsedBridgeVlans := gjson.Parse(bridgeVlans)
			gjsonExpression := linuxbridge.BuildGJsonExpression(bridgeVlans)
			for expectedVlan := minVlan; expectedVlan <= maxVlan; expectedVlan++ {
				vlanByIdAndConection := fmt.Sprintf(gjsonExpression, connection, expectedVlan)
				if !parsedBridgeVlans.Get(vlanByIdAndConection).Exists() {
					return fmt.Errorf("bridge connection %s has no vlan %d, obtainedVlans: \n %s", connection, expectedVlan, bridgeVlans)
				}
			}
		}
		return nil
	}, ReadTimeout, ReadInterval)
}

func vlansCardinality(node string, connection string) AsyncAssertion {
	Byf("Getting vlan cardinality for node %s connection %s", node, connection)
	return Eventually(func() (int, error) {
		bridgeVlans, err := bridgeVlansAtNode(node)
		if err != nil {
			return 0, err
		}

		return len(gjson.Parse(bridgeVlans).Get(connection).Array()), nil
	}, ReadTimeout, ReadInterval)

}

func bridgeDescription(node string, bridgeName string) AsyncAssertion {
	return Eventually(func() (string, error) {
		return runner.RunAtNode(node, "sudo", "ip", "-d", "link", "show", "type", "bridge", bridgeName)
	}, ReadTimeout, ReadInterval)
}

func nextBridge() string {
	bridgeCounter++
	return fmt.Sprintf("br%d", bridgeCounter)
}

func nextBond() string {
	bridgeCounter++
	return fmt.Sprintf("bond%d", bondConunter)
}

func currentStateJSON(node string) []byte {
	key := types.NamespacedName{Name: node}
	currentState := nodeNetworkState(key).Status.CurrentState
	currentStateJson, err := yaml.YAMLToJSON(currentState.Raw)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return currentStateJson
}

func dhcpFlag(node string, name string) bool {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.dhcp", name)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).Bool()
}

func autoDNS(node string, name string) bool {
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

// return a json with all node interfaces and their state e.g.
//{"cni0":"up","docker0":"up","eth0":"up","eth1":"down","eth2":"down","lo":"down"}
// use exclude to filter out interfaces you don't care about
func nodeInterfacesState(node string, exclude []string) []byte {
	var currentStateYaml nmstate.State
	currentState(node, &currentStateYaml).ShouldNot(BeEmpty())

	interfaces := interfaces(currentStateYaml)
	ifacesState := make(map[string]string)
	for _, iface := range interfaces {
		name, hasName := iface.(map[string]interface{})["name"]
		Expect(hasName).To(BeTrue(), "should have name field in the interfaces, https://github.com/nmstate/nmstate/blob/base/libnmstate/schemas/operational-state.yaml")
		if ifaceInSlice(name.(string), exclude) {
			continue
		}
		state, hasState := iface.(map[string]interface{})["state"]
		if !hasState {
			state = "unknown"
		}
		ifacesState[name.(string)] = state.(string)
	}
	ret, err := json.Marshal(ifacesState)
	if err != nil {
		return []byte{}
	}
	return ret
}

func ipv4Address(node string, iface string) string {
	path := fmt.Sprintf("interfaces.#(name==\"%s\").ipv4.address.0.ip", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
}

func defaultRouteNextHopInterface(node string) AsyncAssertion {
	return Eventually(func() string {
		path := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-interface"
		return gjson.ParseBytes(currentStateJSON(node)).Get(path).String()
	}, 15*time.Second, 1*time.Second)
}

func vlan(node string, iface string) string {
	vlanFilter := fmt.Sprintf("interfaces.#(name==\"%s\").vlan.id", iface)
	return gjson.ParseBytes(currentStateJSON(node)).Get(vlanFilter).String()
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
	m, _ := nmstatenode.ScaledMaxUnavailableNodeCount(len(nodes), intstr.FromString(nmstatenode.DEFAULT_MAXUNAVAILABLE))
	return m
}
