package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"

	"github.com/gobwas/glob"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const nmstateCommand = "nmstatectl"

var (
	interfacesFilterGlob      glob.Glob
	interfacesFilterGlobIsSet bool
)

func show(arguments ...string) (string, error) {
	cmd := exec.Command(nmstateCommand, "show")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute nmstatectl show: '%v', '%s', '%s'", err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func applyVlanFiltering(bridgeName string, ports []string) (string, error) {
	command := []string{bridgeName}
	command = append(command, ports...)

	cmd := exec.Command("vlan-filtering", command...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute vlan-filtering: '%v', '%s', '%s'", err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func set(state string) (string, error) {
	cmd := exec.Command(nmstateCommand, "set")
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe for writing into nmstate: %v", err)
	}
	go func() {
		defer stdin.Close()
		_, err = io.WriteString(stdin, state)
		if err != nil {
			fmt.Printf("failed to write state into stdin: %v\n", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute nmstate set: '%v' '%s' '%s'", err, stdout.String(), stderr.String())
	}

	return stdout.String(), nil
}

func GetNodeNetworkState(client client.Client, nodeName string) (nmstatev1alpha1.NodeNetworkState, error) {
	var nodeNetworkState nmstatev1alpha1.NodeNetworkState
	nodeNetworkStateKey := types.NamespacedName{
		Name: nodeName,
	}
	err := client.Get(context.TODO(), nodeNetworkStateKey, &nodeNetworkState)
	return nodeNetworkState, err
}

func InitializeNodeNeworkState(client client.Client, nodeName string) error {
	nodeNetworkState := nmstatev1alpha1.NodeNetworkState{
		// Create NodeNetworkState for this node
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: nmstatev1alpha1.NodeNetworkStateSpec{
			NodeName: nodeName,
		},
	}
	err := client.Create(context.TODO(), &nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error creating NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return nil
}

func UpdateCurrentState(client client.Client, nodeNetworkState *nmstatev1alpha1.NodeNetworkState) error {
	currentState, err := show()
	if err != nil {
		return fmt.Errorf("error running nmstatectl show: %v", err)
	}

	filteredState, err := filterOut(nmstatev1alpha1.State(currentState))
	if err != nil {
		return fmt.Errorf("error filtering out interfaces from NodeNetworkState: %v", err)
	}

	nodeNetworkState.Status.CurrentState = filteredState

	err = client.Status().Update(context.Background(), nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error updating status of NodeNetworkState: %v", err)
	}

	return nil
}

func ApplyDesiredState(nodeNetworkState *nmstatev1alpha1.NodeNetworkState) (string, error) {
	desiredState := string(nodeNetworkState.Spec.DesiredState)
	if len(desiredState) == 0 {
		return "Ignoring empty desired state", nil
	}

	setOutput, err := set(string(nodeNetworkState.Spec.DesiredState))
	if err != nil {
		return setOutput, err
	}

	// Future versions of nmstate/NM will support vlan-filtering meanwhile
	// we have to enforce it at the desiredState bridges and outbound ports
	// they will be configured with vlan_filtering 1 and all the vlan id range
	// set
	// TODO: After implementing commit/rollack from nmstate we have to
	//       rollback if vlanfiltering fails
	bridgesUpWithPorts, err := getBridgesUp(nodeNetworkState.Spec.DesiredState)
	if err != nil {
		return "", fmt.Errorf("error retrieving up bridges from desired state: %v", err)
	}

	commandOutput := ""
	for bridge, ports := range bridgesUpWithPorts {
		outputVlanFiltering, err := applyVlanFiltering(bridge, ports)
		commandOutput += fmt.Sprintf("bridge %s ports %v applyVlanFiltering command output: %s\n", bridge, ports, outputVlanFiltering)
		if err != nil {
			return commandOutput, err
		}
	}

	commandOutput += fmt.Sprintf("setOutput: %s \n", setOutput)
	return commandOutput, nil
}

func getFilter() *glob.Glob {
	if !interfacesFilterGlobIsSet {
		interfacesFilter := os.Getenv("INTERFACES_FILTER")
		interfacesFilterGlob = glob.MustCompile(interfacesFilter)
		interfacesFilterGlobIsSet = true
	}
	return &interfacesFilterGlob
}

func filterOut(currentState nmstatev1alpha1.State) (nmstatev1alpha1.State, error) {
	interfacesFilterGlob := getFilter()

	if (*interfacesFilterGlob).Match("") {
		return currentState, nil
	}

	var state map[string]interface{}
	err := yaml.Unmarshal([]byte(currentState), &state)
	if err != nil {
		return currentState, err
	}

	interfaces := state["interfaces"]
	var filteredInterfaces []interface{}

	for _, iface := range interfaces.([]interface{}) {
		name := iface.(map[string]interface{})["name"]
		if !(*interfacesFilterGlob).Match(name.(string)) {
			filteredInterfaces = append(filteredInterfaces, iface)
		}
	}

	state["interfaces"] = filteredInterfaces
	filteredState, err := yaml.Marshal(state)
	if err != nil {
		return currentState, err
	}

	return filteredState, nil
}
