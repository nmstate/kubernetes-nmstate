package nmstatectl

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"github.com/nmstate/k8s-node-net-conf/pkg/apis/nmstate.io/v1"
)

const nmstateCommand = "nmstatectl"

// Show is populating the passed ConfAndOperationalState object from the output of "nmstatectl show" 
func Show(currentState *v1.ConfAndOperationalState) error {
	cmd := exec.Command(nmstateCommand, "show")

	if buff, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to execute nmstate: '%v'\n'%s'\n ", err, string(buff))
	} else {
		if err = json.Unmarshal(buff, currentState); err != nil {
			fmt.Printf("Failed to decode JSON output: %v\n", err)
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
		fmt.Printf("%s\n", out)
		fmt.Printf("Failed to execute nmstate: %v\n", err)
		return err
	}

	return nil
}
