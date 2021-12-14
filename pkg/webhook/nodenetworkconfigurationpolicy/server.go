package nodenetworkconfigurationpolicy

import (
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func Add(mgr manager.Manager) error {
	// We need two hooks, the update of nncp and nncp/status (it's a subresource) happens
	// at different times, also if you modify status at nncp webhook it does not modify it,
	// so you need nncp/status webhook that will catch that and do the final modifications.
	// So this works this way:
	// 1.- User changes nncp desiredState so it triggers deleteConditionsHook()
	// 2.- Since we have deleted the condition the status-mutate webhook is called and
	//     there we set conditions to Unknown. This final result will be updated.
	server := &webhook.Server{}
	server.Register("/readyz", healthz.CheckHandler{Checker: healthz.Ping})
	server.Register("/nodenetworkconfigurationpolicies-mutate", deleteConditionsHook())
	server.Register("/nodenetworkconfigurationpolicies-status-mutate", setConditionsUnknownHook())
	server.Register("/nodenetworkconfigurationpolicies-timestamp-mutate", setTimestampAnnotationHook())
	server.Register("/nodenetworkconfigurationpolicies-update-validate", validatePolicyUpdateHook(mgr.GetClient()))
	server.Register("/nodenetworkconfigurationpolicies-create-validate", validatePolicyCreateHook(mgr.GetClient()))
	return mgr.Add(server)
}
