package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

const nmstateCommand = "nmstatectl"

func show(arguments ...string) (string, error) {
	cmd := exec.Command(nmstateCommand, "show")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to execute nmstatectl show: '%v', %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func set(state string) error {
	cmd := exec.Command(nmstateCommand, "set")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("Failed to create pipe for writing  into nmstate: %v", err)
	}
	defer stdin.Close()
	_, err = io.WriteString(stdin, state)
	if err != nil {
		return fmt.Errorf("Failed to write state into stdin: %v", err)
	}
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute nmstate set: '%v' '%s'", err, stderr.String())
	}
	return nil
}

func GetNodeNetworkState(client client.Client, nodeName string) (nmstatev1.NodeNetworkState, error) {
	var nodeNetworkState nmstatev1.NodeNetworkState
	nodeNetworkStateKey := types.NamespacedName{
		Name: nodeName,
	}
	err := client.Get(context.TODO(), nodeNetworkStateKey, &nodeNetworkState)
	return nodeNetworkState, err
}

func InitializeNodeNeworkState(client client.Client, nodeName string) error {
	nodeNetworkState := nmstatev1.NodeNetworkState{
		// Create NodeNetworkState for this node
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: nmstatev1.NodeNetworkStateSpec{
			NodeName: nodeName,
		},
	}
	err := client.Create(context.TODO(), &nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error creating NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return nil
}

func UpdateCurrentState(client client.Client, nodeNetworkState *nmstatev1.NodeNetworkState) error {
	currentState, err := show()
	if err != nil {
		return fmt.Errorf("Error running nmstatectl show: %v", err)
	}

	// Let's update status with current network config from nmstatectl
	nodeNetworkState.Status = nmstatev1.NodeNetworkStateStatus{
		CurrentState: nmstatev1.State(currentState),
	}

	err = client.Status().Update(context.Background(), nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error updating status of NodeNetworkState: %v", err)
	}

	return nil
}

func ApplyDesiredState(nodeNetworkState *nmstatev1.NodeNetworkState) error {
	return set(string(nodeNetworkState.Spec.DesiredState))
}
