package helper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	yaml "sigs.k8s.io/yaml"

	"github.com/gobwas/glob"
	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/probe"
)

var (
	log = logf.Log.WithName("client")
)

const vlanFilteringCommand = "vlan-filtering"
const defaultGwRetrieveTimeout = 120 * time.Second
const defaultGwProbeTimeout = 120 * time.Second
const apiServerProbeTimeout = 120 * time.Second

var (
	interfacesFilterGlob glob.Glob
)

func init() {
	if !environment.IsHandler() {
		return
	}
	interfacesFilter, isSet := os.LookupEnv("INTERFACES_FILTER")
	if !isSet {
		panic("INTERFACES_FILTER is mandatory")
	}
	interfacesFilterGlob = glob.MustCompile(interfacesFilter)
}

func applyVlanFiltering(bridgeName string, ports []string) (string, error) {
	command := []string{bridgeName}
	command = append(command, ports...)

	cmd := exec.Command(vlanFilteringCommand, command...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s: '%v', '%s', '%s'", vlanFilteringCommand, err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func GetNodeNetworkState(client client.Client, nodeName string) (nmstatev1beta1.NodeNetworkState, error) {
	var nodeNetworkState nmstatev1beta1.NodeNetworkState
	nodeNetworkStateKey := types.NamespacedName{
		Name: nodeName,
	}
	err := client.Get(context.TODO(), nodeNetworkStateKey, &nodeNetworkState)
	return nodeNetworkState, err
}

func InitializeNodeNetworkState(client client.Client, node *corev1.Node) error {
	ownerRefList := []metav1.OwnerReference{{Name: node.ObjectMeta.Name, Kind: "Node", APIVersion: "v1", UID: node.UID}}

	nodeNetworkState := nmstatev1beta1.NodeNetworkState{
		// Create NodeNetworkState for this node
		ObjectMeta: metav1.ObjectMeta{
			Name:            node.ObjectMeta.Name,
			OwnerReferences: ownerRefList,
		},
	}

	err := client.Create(context.TODO(), &nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error creating NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return nil
}

func CreateOrUpdateNodeNetworkState(client client.Client, node *corev1.Node, namespace client.ObjectKey) error {
	nnsInstance := &nmstatev1beta1.NodeNetworkState{}
	err := client.Get(context.TODO(), namespace, nnsInstance)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "Failed to get nmstate")
		} else {
			return InitializeNodeNetworkState(client, node)
		}
	}
	return UpdateCurrentState(client, nnsInstance)
}

func UpdateCurrentState(client client.Client, nodeNetworkState *nmstatev1beta1.NodeNetworkState) error {
	observedStateRaw, err := nmstatectl.Show()
	if err != nil {
		return errors.Wrap(err, "error running nmstatectl show")
	}
	observedState := nmstate.State{Raw: []byte(observedStateRaw)}

	stateToReport, err := filterOut(observedState, interfacesFilterGlob)
	if err != nil {
		fmt.Printf("failed filtering out interfaces from NodeNetworkState, keeping orignal content, please fix the glob: %v", err)
		stateToReport = observedState
	}

	nodeNetworkState.Status.CurrentState = stateToReport
	nodeNetworkState.Status.LastSuccessfulUpdateTime = metav1.Time{Time: time.Now()}

	err = client.Status().Update(context.Background(), nodeNetworkState)
	if err != nil {
		// Request object not found, could have been deleted after reconcile request.
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "Request object not found, could have been deleted after reconcile request")
		}
	}

	return nil
}

func rollback(client client.Client, cause error) error {
	message := fmt.Sprintf("rolling back desired state configuration: %s", cause)
	err := nmstatectl.Rollback()
	if err != nil {
		return errors.Wrap(err, message)
	}

	// wait for system to settle after rollback
	probesErr := probe.RunAll(client)
	if probesErr != nil {
		return errors.Wrap(errors.Wrap(err, "failed running probes after rollback"), message)
	}
	return errors.New(message)
}

func ApplyDesiredState(client client.Client, desiredState nmstate.State) (string, error) {
	if len(string(desiredState.Raw)) == 0 {
		return "Ignoring empty desired state", nil
	}

	// commit timeout doubles the default gw ping probe and check API server
	// connectivity timeout, to
	// ensure the Checkpoint is alive before rolling it back
	// https://nmstate.github.io/cli_guide#manual-transaction-control
	setOutput, err := nmstatectl.Set(desiredState, (defaultGwProbeTimeout+apiServerProbeTimeout)*2)
	if err != nil {
		return setOutput, err
	}

	// Future versions of nmstate/NM will support vlan-filtering meanwhile
	// we have to enforce it at the desiredState bridges and outbound ports
	// they will be configured with vlan_filtering 1 and all the vlan id range
	// set
	bridgesUpWithPorts, err := getBridgesUp(desiredState)
	if err != nil {
		return "", rollback(client, fmt.Errorf("error retrieving up bridges from desired state"))
	}

	commandOutput := ""
	for bridge, ports := range bridgesUpWithPorts {
		outputVlanFiltering, err := applyVlanFiltering(bridge, ports)
		commandOutput += fmt.Sprintf("bridge %s ports %v applyVlanFiltering command output: %s\n", bridge, ports, outputVlanFiltering)
		if err != nil {
			return commandOutput, rollback(client, err)
		}
	}

	err = probe.RunAll(client)
	if err != nil {
		return "", rollback(client, errors.Wrap(err, "failed runnig probes after network changes"))
	}

	commitOutput, err := nmstatectl.Commit()
	if err != nil {
		// We cannot rollback if commit fails, just return the error
		return commitOutput, err
	}

	commandOutput += fmt.Sprintf("setOutput: %s \n", setOutput)
	return commandOutput, nil
}

func filterOut(currentState nmstate.State, interfacesFilterGlob glob.Glob) (nmstate.State, error) {
	if interfacesFilterGlob.Match("") {
		return currentState, nil
	}

	var state map[string]interface{}
	err := yaml.Unmarshal(currentState.Raw, &state)
	if err != nil {
		return currentState, err
	}

	interfaces := state["interfaces"]
	var filteredInterfaces []interface{}

	for _, iface := range interfaces.([]interface{}) {
		name := iface.(map[string]interface{})["name"]
		if !interfacesFilterGlob.Match(name.(string)) {
			filteredInterfaces = append(filteredInterfaces, iface)
		}
	}

	state["interfaces"] = filteredInterfaces
	filteredState, err := yaml.Marshal(state)
	if err != nil {
		return currentState, err
	}

	return nmstate.State{Raw: filteredState}, nil
}
