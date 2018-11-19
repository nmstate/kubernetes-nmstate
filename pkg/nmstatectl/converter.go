package nmstatectl

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"github.com/nmstate/k8s-node-net-conf/pkg/apis/nmstate.io/v1"
	nmstatev1 "github.com/nmstate/k8s-node-net-conf/pkg/client/clientset/versioned/typed/nmstate.io/v1"
)

const nmstateCommand = "nmstatectl"

// Show is populating the passed ConfAndOperationalState object from the output of "nmstatectl show" 
func Show(currentState *v1.ConfAndOperationalState) error {
	cmd := exec.Command(nmstateCommand, "show")

	if buff, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to execute nmstate: '%v'\n'%s'\n ", err, string(buff))
	} else {
		if err = json.Unmarshal(buff, currentState); err != nil {
			fmt.Printf("ERROR: %s\n", string(buff))
			fmt.Printf("Failed to decode JSON output: %v\n", err)
		} else {
			fmt.Printf("DEBUG: %s\n", string(buff))
		}
	}

	return nil
}


// Set is executing "nmstatectl set" based on the parameters passed in the ConfigurationState object
func Set(desiredState *v1.ConfigurationState) error {
	cmd := exec.Command(nmstateCommand, "set")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Failed to create pipe for writing into nmstate: %v\n", err)
	}

	if err := json.NewEncoder(stdin).Encode(desiredState); err != nil {
		fmt.Printf("Failed to encode JSON input: %v\n", err)
		return err
	}
	stdin.Close()

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("ERROR: %s\n", out)
		fmt.Printf("Failed to execute nmstate: %v\n", err)
		return err
	} else {
		fmt.Printf("DEBUG: %s\n", out)
	}

	return nil
}

// HandleResource is used for handling of NodeNetworkState CRDs
func HandleResource(state *v1.NodeNetworkState, client nmstatev1.NmstateV1Interface) (err error) {
	if state.Spec.Managed {
		if err = Set(&state.Spec.DesiredState); err != nil {
			fmt.Printf("Failed set state on node: %v\n", err)
		}
	} else {
		fmt.Printf("Node '%s' is unmanaged by state '%s'\n", state.Spec.NodeName, state.Name)
	}

	// TODO: should we update current state for unmanaged nodes?
	if err = Show(&state.Status.CurrentState); err != nil {
		fmt.Printf("Failed to fetch current state: %v\n", err)
	} else {
		if _, err := client.NodeNetworkStates(state.Namespace).Update(state); err != nil {
			fmt.Printf("Failed to update state: %v\n", err)
		} else {
			fmt.Printf("Successfully update state '%s' on node '%s'\n", state.Name, state.Spec.NodeName)
		}
	}

	return
}
