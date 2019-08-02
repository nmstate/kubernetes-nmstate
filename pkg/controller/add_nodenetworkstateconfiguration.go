package controller

import (
	"github.com/nmstate/kubernetes-nmstate/pkg/controller/nodenetworkstateconfiguration"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nodenetworkstateconfiguration.Add)
}
