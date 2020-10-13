package controller

import (
	"github.com/nmstate/kubernetes-nmstate/controllers/node"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
)

func init() {
	if !environment.IsHandler() {
		return
	}

	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, node.Add)
}
