package nodenetworkconfigurationpolicy

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pkg/errors"

	"github.com/qinqon/kube-admission-webhook/pkg/certificate"
	webhookserver "github.com/qinqon/kube-admission-webhook/pkg/webhook/server"
)

const (
	webhookName = "nmstate"
)

func Add(mgr manager.Manager, o certificate.Options) error {

	// We need two hooks, the update of nncp and nncp/status (it's a subresource) happens
	// at different times, also if you modify status at nncp webhook it does not modify it
	// you need at nncp/status webhook that will catch that and do the final modifications.
	// So this works this way:
	// 1.- User changes nncp desiredState so it triggers deleteConditionsHook()
	// 2.- Since we have delete the condition the status-mutate webhook get called and
	//     there we set conditions to Unknown this final result will be updated.
	server, err := webhookserver.New(mgr.GetClient(), o,
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-mutate", deleteConditionsHook()),
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-status-mutate", setConditionsUnknownHook()),
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-timestamp-mutate", setTimestampAnnotationHook()),
	)
	if err != nil {
		return errors.Wrap(err, "failed creating new webhook server")
	}
	return server.Add(mgr)
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s manager.Runnable) error {
	mgr.Add(s)
	return nil
}
