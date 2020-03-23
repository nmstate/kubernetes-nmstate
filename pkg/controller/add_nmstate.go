package controller

import (
	"os"

	"github.com/nmstate/kubernetes-nmstate/pkg/controller/nmstate"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	if _, runOperator := os.LookupEnv("RUN_OPERATOR"); runOperator {
		AddToManagerFuncs = append(AddToManagerFuncs, nmstate.Add)
	}
}
