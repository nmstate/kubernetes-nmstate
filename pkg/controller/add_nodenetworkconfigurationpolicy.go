package controller

import (
	"github.com/nmstate/kubernetes-nmstate/pkg/controller/nodenetworkconfigurationpolicy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nodenetworkconfigurationpolicy.Add)
}
