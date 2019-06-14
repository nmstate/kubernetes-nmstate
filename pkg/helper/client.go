package helper

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

const nmstateCommand = "nmstatectl"
const namespace = "default"

func nmstatectl(arguments ...string) (string, error) {
	//cmd := exec.Command(nmstateCommand, arguments...)
	cmd := exec.Command(nmstateCommand, arguments...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to execute nmstatectl show: '%v'", err)
	}
	return outb.String(), nil
}

func GetNodeNetworkState(client client.Client, nodeName string, nodeNetworkState *nmstatev1.NodeNetworkState) error {

	nodeNetworkStateKey := types.NamespacedName{
		Namespace: namespace,
		Name:      nodeName,
	}

	return client.Get(context.TODO(), nodeNetworkStateKey, nodeNetworkState)
}

func CreateNodeNeworkState(client client.Client, nodeName string) error {

	nodeNetworkState := nmstatev1.NodeNetworkState{}
	// Create NodeNetworkState for this node
	nodeNetworkState.ObjectMeta = metav1.ObjectMeta{
		Name:      nodeName,
		Namespace: namespace,
	}
	nodeNetworkState.Spec = nmstatev1.NodeNetworkStateSpec{
		NodeName: nodeName,
	}
	// There is no NodeNetworkState for this node let's create it
	err := client.Create(context.TODO(), &nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error creating NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return nil
}

func DeleteNodeNetworkState(client client.Client, nodeNetworkState *nmstatev1.NodeNetworkState) error {

	// There is no NodeNetworkState for this node let's create it
	err := client.Delete(context.TODO(), nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error deleting NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return nil
}

func UpdateCurrentState(client client.Client, nodeNetworkState *nmstatev1.NodeNetworkState) error {
	currentState, err := nmstatectl("show")
	if err != nil {
		return fmt.Errorf("Error running nmstatectl show: %v", err)
	}

	// Let's update status with current network config from nmstatectl
	nodeNetworkState.Status = nmstatev1.NodeNetworkStateStatus{
		CurrentState: nmstatev1.State(currentState),
	}

	err = client.Status().Update(context.TODO(), nodeNetworkState)
	if err != nil {
		return fmt.Errorf("error updating status of NodeNetworkState: %v", err)
	}

	return nil
}
