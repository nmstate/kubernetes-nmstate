package nmstatectl

import (
	"encoding/json"
	"fmt"
	"github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned/typed/nmstate.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
)

const nmstateCommand = "nmstatectl"

func Check() error {
	cmd := exec.Command(nmstateCommand, "show")
	if buff, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s\n%v", string(buff), err)
	}
	return nil
}

// Show is populating the passed ConfAndOperationalState object from the output of "nmstatectl show"
func Show(currentState *v1.ConfAndOperationalState) (err error) {
	cmd := exec.Command(nmstateCommand, "show", "--json")
	var buff []byte

	if buff, err = cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to execute nmstatectl show: '%v'\n'%s'\n ", err, string(buff))
		return
	}

	if err = json.Unmarshal(buff, currentState); err != nil {
		fmt.Printf("Failed to decode JSON output: %v\n", err)
		return
	}

	return nil
}

// Set is executing "nmstatectl set" based on the parameters passed in the ConfigurationState object
func Set(desiredState *v1.ConfigurationState) error {
	cmd := exec.Command(nmstateCommand, "set")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Failed to create pipe for writing into nmstate: %v\n", err)
		return err
	}

	if err := json.NewEncoder(stdin).Encode(desiredState); err != nil {
		fmt.Printf("Failed to encode JSON input: %v\n", err)
		return err
	}
	stdin.Close()

	if buff, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to execute nmstatectl set: '%v'\n'%s'\n", err, string(buff))
		return err
	}

	return nil
}

// convertErrorToStatus convert the error reply from nmstate into a status object
// in the NodeNetworkState CRD
func convertErrorToStatus(err error) {
	// TODO
}

// HandleResource is used for handling of NodeNetworkState CRDs
func HandleResource(state *v1.NodeNetworkState, client nmstatev1.NmstateV1Interface) (*v1.NodeNetworkState, error) {
	if !state.Spec.Managed {
		fmt.Printf("Node '%s' is unmanaged by state '%s'\n", state.Spec.NodeName, state.Name)
		return nil, nil
	}

	var setErr error
	// check if resource has any desired state
	if len(state.Spec.DesiredState.Interfaces) > 0 {
		if setErr = Set(&state.Spec.DesiredState); setErr != nil {
			fmt.Printf("Failed set state on node '%s': %v\n", state.Spec.NodeName, setErr)
			// still try to update current state, but return the set error
		}
	} else {
		fmt.Printf("No configuration state to set on node '%s'\n", state.Spec.NodeName)
	}

	if err := Show(&state.Status.CurrentState); err != nil {
		fmt.Printf("Failed to fetch current state: %v\n", err)
		if setErr != nil {
			err = setErr
		}
		return nil, err
	}

	if newState, err := client.NodeNetworkStates(state.Namespace).Update(state); err != nil {
		fmt.Printf("Failed to update state: %v\n", err)
		if setErr != nil {
			err = setErr
		}
		return nil, err
	} else {
		fmt.Printf("Successfully update state '%s' on node '%s'\n", state.Name, state.Spec.NodeName)
		return newState, setErr
	}
}

// CreateResource is used for creating of NodeNetworkState CRDs
func CreateResource(client nmstatev1.NmstateV1Interface, name string, namespace string) (*v1.NodeNetworkState, error) {
	state := &v1.NodeNetworkState{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.SchemeGroupVersionNodeNetworkState.Kind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.NodeNetworkStateSpec{
			Managed:  true,
			NodeName: name,
		},
		Status: v1.NodeNetworkStateStatus{CurrentState: v1.ConfAndOperationalState{}},
	}

	if err := Show(&state.Status.CurrentState); err != nil {
		fmt.Printf("Failed to fetch current state: %v\n", err)
		return nil, err
	}

	if _, err := client.NodeNetworkStates(state.Namespace).Create(state); err != nil {
		fmt.Printf("Failed to create state: %v\n", err)
		return nil, err
	}

	fmt.Printf("Successfully created state '%s' on node '%s'\n", state.Name, state.Spec.NodeName)
	return state, nil
}
