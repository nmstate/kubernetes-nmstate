package nodenetworkconfigurationpolicy

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	webhookName = "nmstate"
)

func Add(mgr manager.Manager) error {
	// We need two hooks, the update of nncp and nncp/status (it's a subresource) happends
	// at different times, also if you modify status at nncp webhook it does not modify it
	// you need at nncp/status webhook that will catch that and do the final modifications.
	// So this works this way:
	// 1.- User changes nncp desiredState so it triggers deleteConditionsHook()
	// 2.- Since we have delete the condition the status-mutate webhook get called and
	//     there we set conditions to Unknown this final result will be updated.
	server := &webhook.Server{}
	server.Register("/nodenetworkconfigurationpolicies-mutate", deleteConditionsHook())
	server.Register("/nodenetworkconfigurationpolicies-status-mutate", setConditionsUnknownHook())
	server.Register("/nodenetworkconfigurationpolicies-timestamp-mutate", setTimestampAnnotationHook())
	server.Register("/nodenetworkconfigurationpolicies-progress-validate", validatePolicyUpdateHook(mgr.GetClient()))
	return mgr.Add(server)
}
