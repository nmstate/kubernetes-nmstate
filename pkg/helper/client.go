package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"

	"github.com/gobwas/glob"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const nmstateCommand = "nmstatectl"
const vlanFilteringCommand = "vlan-filtering"

var (
	interfacesFilterGlob glob.Glob
)

func init() {
	interfacesFilter, isSet := os.LookupEnv("INTERFACES_FILTER")
	if !isSet {
		panic("INTERFACES_FILTER is mandatory")
	}
	interfacesFilterGlob = glob.MustCompile(interfacesFilter)
}

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

	cmd := exec.Command(vlanFilteringCommand, command...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s: '%v', '%s', '%s'", vlanFilteringCommand, err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}

func nmstatectl(arguments []string, input string) (string, error) {
	cmd := exec.Command(nmstateCommand, arguments...)
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("failed to create pipe for writing into %s: %v", nmstateCommand, err)
		}
		go func() {
			defer stdin.Close()
			_, err = io.WriteString(stdin, input)
			if err != nil {
				fmt.Printf("failed to write input into stdin: %v\n", err)
			}
		}()

	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s %s: '%v' '%s' '%s'", nmstateCommand, strings.Join(arguments, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil

}

func set(state string) (string, error) {
	output := ""
	var err error = nil
	// FIXME: Remove this retries after nmstate team fixes
	//        https://nmstate.atlassian.net/browse/NMSTATE-247
	retries := 2
	for retries > 0 {
		output, err = nmstatectl([]string{"set", "--no-commit"}, state)
		if err == nil {
			break
		}
		retries--
	}
	return output, err
}

func commit() (string, error) {
	return nmstatectl([]string{"commit"}, "")
}

func rollback(cause error) error {
	_, err := nmstatectl([]string{"rollback"}, "")
	return fmt.Errorf("rollback cause: %v, rollback error: %v", cause, err)
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
	observedStateRaw, err := show()
	if err != nil {
		return fmt.Errorf("error running nmstatectl show: %v", err)
	}
	observedState := nmstatev1alpha1.State(observedStateRaw)

	stateToReport, err := filterOut(observedState, interfacesFilterGlob)
	if err != nil {
		fmt.Printf("failed filtering out interfaces from NodeNetworkState, keeping orignal content, please fix the glob: %v", err)
		stateToReport = observedState
	}

	nodeNetworkState.Status.CurrentState = stateToReport

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
	bridgesUpWithPorts, err := getBridgesUp(nodeNetworkState.Spec.DesiredState)
	if err != nil {
		return "", rollback(fmt.Errorf("error retrieving up bridges from desired state"))
	}

	commandOutput := ""
	for bridge, ports := range bridgesUpWithPorts {
		outputVlanFiltering, err := applyVlanFiltering(bridge, ports)
		commandOutput += fmt.Sprintf("bridge %s ports %v applyVlanFiltering command output: %s\n", bridge, ports, outputVlanFiltering)
		if err != nil {
			return commandOutput, rollback(err)
		}
	}

	_, err = commit()
	if err != nil {
		return commandOutput, rollback(err)
	}
	commandOutput += fmt.Sprintf("setOutput: %s \n", setOutput)
	return commandOutput, nil
}

func filterOut(currentState nmstatev1alpha1.State, interfacesFilterGlob glob.Glob) (nmstatev1alpha1.State, error) {
	if interfacesFilterGlob.Match("") {
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
		if !interfacesFilterGlob.Match(name.(string)) {
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
