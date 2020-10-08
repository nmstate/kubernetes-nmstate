package controller

import (
	"github.com/nmstate/kubernetes-nmstate/controllers/nodenetworkconfigurationpolicy"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
)

func init() {
	if !environment.IsHandler() {
		return
	}

	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nodenetworkconfigurationpolicy.Add)
}
