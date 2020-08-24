package webhook

import (
	"github.com/qinqon/kube-admission-webhook/pkg/certificate"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(m manager.Manager, o certificate.Options) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, o certificate.Options) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, o); err != nil {
			return err
		}
	}
	return nil
}
