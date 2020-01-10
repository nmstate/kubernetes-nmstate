package webhook

import (
	"github.com/nmstate/kubernetes-nmstate/pkg/webhook/mutating"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, mutating.Add)
}
